package newtqnia_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	newtqnia "github.com/newtqnia/newtqnia-go"
)

const digestJSON = `{"api_version":"v1","collection":"latest","timezone":"Asia/Dubai","locale":"en","direction":"ltr","publisher":{"name":"NewTqnia","url":"https://newtqnia.com"},"attribution":{"text":"Powered by NewTqnia","url":"https://newtqnia.com","required":true},"generated_at":"2026-08-21T12:00:00+04:00","articles":[{"id":7,"title":"Story","summary":"Summary","category":{"slug":"ai","name":"AI"},"image":"https://example.com/a.webp","url":"https://newtqnia.com/en/news/7-story","published_at":"2026-08-21T11:00:00+04:00","read_time":3}],"_links":{}}`

func TestLatestSerializesAndAuthenticates(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.RawQuery; got != "category=AI+%26+ML&limit=5&locale=ar" {
			t.Errorf("query = %q", got)
		}
		if r.Header.Get("X-API-Key") != "secret" || r.Header.Get("X-NewTqnia-Application") != "Dashboard" || r.Header.Get("X-NewTqnia-Website") != "https://example.com" {
			t.Error("identity headers missing")
		}
		if r.Header.Get("X-Test") != "yes" || r.Header.Get("User-Agent") != "test/1" {
			t.Error("custom headers missing")
		}
		_, _ = io.WriteString(w, strings.Replace(digestJSON, `"locale":"en"`, `"locale":"ar"`, 1))
	}))
	defer server.Close()
	client, err := newtqnia.NewClient(newtqnia.Config{BaseURL: server.URL, APIKey: "secret", Application: "Dashboard", Website: "https://example.com", UserAgent: "test/1", Headers: http.Header{"X-Test": {"yes"}}})
	if err != nil {
		t.Fatal(err)
	}
	digest, err := client.News.Latest(context.Background(), &newtqnia.NewsListParams{Locale: newtqnia.LocaleArabic, Limit: 5, Category: "AI & ML"})
	if err != nil {
		t.Fatal(err)
	}
	if digest.Locale != newtqnia.LocaleArabic || len(digest.Articles) != 1 || digest.Articles[0].Title != "Story" {
		t.Fatalf("unexpected digest: %#v", digest)
	}
}

func TestRetriesTemporaryResponses(t *testing.T) {
	var calls atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) == 1 {
			http.Error(w, "busy", http.StatusServiceUnavailable)
			return
		}
		_, _ = io.WriteString(w, digestJSON)
	}))
	defer server.Close()
	client, _ := newtqnia.NewClient(newtqnia.Config{BaseURL: server.URL, Retry: newtqnia.RetryConfig{MaxRetries: 1, MaxWait: time.Millisecond}})
	if _, err := client.News.Latest(context.Background(), nil); err != nil {
		t.Fatal(err)
	}
	if calls.Load() != 2 {
		t.Fatalf("calls = %d", calls.Load())
	}
}

func TestTypedRateLimitError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "2")
		w.Header().Set("X-Request-ID", "req_1")
		w.WriteHeader(429)
		_, _ = io.WriteString(w, `{"message":"Slow down","code":"rate_limited"}`)
	}))
	defer server.Close()
	client, _ := newtqnia.NewClient(newtqnia.Config{BaseURL: server.URL, DisableRetries: true})
	_, err := client.News.Today(context.Background(), nil)
	var rateLimit *newtqnia.RateLimitError
	if !errors.As(err, &rateLimit) {
		t.Fatalf("error = %T %v", err, err)
	}
	if rateLimit.RetryAfter != 2*time.Second || rateLimit.RequestID != "req_1" || rateLimit.Code != "rate_limited" {
		t.Fatalf("metadata = %#v", rateLimit)
	}
	var apiErr *newtqnia.APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != 429 {
		t.Fatalf("base API error = %#v", apiErr)
	}
}

func TestValidationAndCancellation(t *testing.T) {
	if _, err := newtqnia.NewClient(newtqnia.Config{BaseURL: "http://example.com"}); err == nil {
		t.Fatal("expected insecure URL error")
	}
	client := newtqnia.New()
	if _, err := client.News.Latest(context.Background(), &newtqnia.NewsListParams{Limit: 11}); err == nil {
		t.Fatal("expected limit error")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.News.Latest(ctx, nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v", err)
	}
}

func TestTimeoutError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		time.Sleep(100 * time.Millisecond)
		_, _ = io.WriteString(w, digestJSON)
	}))
	defer server.Close()
	client, _ := newtqnia.NewClient(newtqnia.Config{BaseURL: server.URL, Timeout: time.Millisecond, DisableRetries: true})
	_, err := client.News.Latest(context.Background(), nil)
	var timeout *newtqnia.TimeoutError
	if !errors.As(err, &timeout) {
		t.Fatalf("error = %T %v", err, err)
	}
}
