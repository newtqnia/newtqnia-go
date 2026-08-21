package newtqnia

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	// Version is the semantic version of this SDK.
	Version          = "1.0.0"
	defaultBaseURL   = "https://api.newtqnia.com"
	defaultUserAgent = "newtqnia-go/" + Version
	maxResponseSize  = 4 << 20
)

// RetryConfig controls retries after the initial request.
type RetryConfig struct {
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// Config configures a Client. Zero values use documented defaults.
type Config struct {
	APIKey      string
	Application string
	Website     string
	BaseURL     string
	Timeout     time.Duration
	Retry       RetryConfig
	// DisableRetries makes the initial request the only attempt.
	DisableRetries bool
	HTTPClient     *http.Client
	UserAgent      string
	Headers        http.Header
}

// Client is a concurrency-safe NewTqnia API client.
type Client struct {
	baseURL     *url.URL
	httpClient  *http.Client
	timeout     time.Duration
	retry       RetryConfig
	apiKey      string
	application string
	website     string
	userAgent   string
	headers     http.Header

	// News exposes localized news collection operations.
	News *NewsService
}

// NewClient constructs a client. The public endpoints require no API key.
func NewClient(config Config) (*Client, error) {
	base := strings.TrimRight(config.BaseURL, "/")
	if base == "" {
		base = defaultBaseURL
	}
	parsed, err := url.Parse(base)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return nil, &ValidationError{Message: "BaseURL must be an absolute HTTP(S) URL"}
	}
	local := parsed.Hostname() == "localhost" || parsed.Hostname() == "127.0.0.1" || parsed.Hostname() == "::1"
	if parsed.Scheme != "https" && !local {
		return nil, &ValidationError{Message: "BaseURL must use HTTPS except for local development"}
	}
	timeout := config.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	if timeout < 0 {
		return nil, &ValidationError{Message: "Timeout must be positive"}
	}
	retry := config.Retry
	if retry.MaxRetries == 0 && retry.InitialWait == 0 && retry.MaxWait == 0 {
		retry = RetryConfig{MaxRetries: 2, InitialWait: 500 * time.Millisecond, MaxWait: 8 * time.Second}
	}
	if config.DisableRetries {
		retry.MaxRetries = 0
	}
	if retry.MaxRetries < 0 || retry.InitialWait < 0 || retry.MaxWait < 0 {
		return nil, &ValidationError{Message: "retry values cannot be negative"}
	}
	if retry.MaxWait == 0 {
		retry.MaxWait = 8 * time.Second
	}
	if retry.InitialWait == 0 && retry.MaxRetries > 0 {
		retry.InitialWait = 500 * time.Millisecond
	}
	httpClient := config.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{}
	}
	userAgent := config.UserAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}
	c := &Client{
		baseURL: parsed, httpClient: httpClient, timeout: timeout, retry: retry,
		apiKey: config.APIKey, application: config.Application, website: config.Website,
		userAgent: userAgent, headers: config.Headers.Clone(),
	}
	c.News = &NewsService{client: c}
	return c, nil
}

// New returns a client using production defaults. It is convenient for callers
// that do not need custom configuration.
func New() *Client {
	c, _ := NewClient(Config{})
	return c
}

// NewsService provides the public news endpoints.
type NewsService struct{ client *Client }

// Today gets articles published today using the Asia/Dubai day boundary.
func (s *NewsService) Today(ctx context.Context, params *NewsListParams) (*Digest, error) {
	return s.get(ctx, CollectionToday, params)
}

// Latest gets the latest published articles.
func (s *NewsService) Latest(ctx context.Context, params *NewsListParams) (*Digest, error) {
	return s.get(ctx, CollectionLatest, params)
}

func (s *NewsService) get(ctx context.Context, collection Collection, params *NewsListParams) (*Digest, error) {
	values := url.Values{}
	locale, limit, category := LocaleEnglish, 10, ""
	if params != nil {
		if params.Locale != "" {
			locale = params.Locale
		}
		if params.Limit != 0 {
			limit = params.Limit
		}
		category = params.Category
	}
	if locale != LocaleEnglish && locale != LocaleArabic {
		return nil, &ValidationError{Message: `Locale must be "en" or "ar"`}
	}
	if limit < 1 || limit > 10 {
		return nil, &ValidationError{Message: "Limit must be between 1 and 10"}
	}
	if params != nil && params.Category != "" && strings.TrimSpace(category) == "" {
		return nil, &ValidationError{Message: "Category cannot be blank"}
	}
	values.Set("locale", string(locale))
	values.Set("limit", strconv.Itoa(limit))
	if category != "" {
		values.Set("category", category)
	}
	path := "/v1/news/" + string(collection) + "?" + values.Encode()
	var digest Digest
	if err := s.client.get(ctx, path, &digest); err != nil {
		return nil, err
	}
	return &digest, nil
}

func (c *Client) get(ctx context.Context, path string, target any) error {
	var lastErr error
	for attempt := 0; attempt <= c.retry.MaxRetries; attempt++ {
		requestCtx, cancel := context.WithTimeout(ctx, c.timeout)
		req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, c.baseURL.String()+path, nil)
		if err != nil {
			cancel()
			return err
		}
		for key, values := range c.headers {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", c.userAgent)
		if c.apiKey != "" {
			req.Header.Set("X-API-Key", c.apiKey)
		}
		if c.application != "" {
			req.Header.Set("X-NewTqnia-Application", c.application)
		}
		if c.website != "" {
			req.Header.Set("X-NewTqnia-Website", c.website)
		}
		resp, err := c.httpClient.Do(req)
		if err == nil {
			if resp.StatusCode >= 200 && resp.StatusCode < 300 {
				err = decodeResponse(resp, target)
				cancel()
				return err
			}
			apiErr := readAPIError(resp)
			if retryable(resp.StatusCode) && attempt < c.retry.MaxRetries {
				wait := apiErr.RetryAfter
				if wait <= 0 {
					wait = c.backoff(attempt)
				}
				cancel()
				if err := sleep(ctx, wait); err != nil {
					return err
				}
				continue
			}
			cancel()
			return classifyAPIError(apiErr)
		}
		cancel()
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if errors.Is(err, context.DeadlineExceeded) || isTimeout(err) {
			lastErr = &TimeoutError{Duration: c.timeout, Err: err}
		} else {
			lastErr = err
		}
		if attempt < c.retry.MaxRetries {
			if err := sleep(ctx, c.backoff(attempt)); err != nil {
				return err
			}
			continue
		}
	}
	var timeout *TimeoutError
	if errors.As(lastErr, &timeout) {
		return timeout
	}
	return &NetworkError{Err: lastErr}
}

func decodeResponse(resp *http.Response, target any) error {
	defer resp.Body.Close()
	decoder := json.NewDecoder(io.LimitReader(resp.Body, maxResponseSize))
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("newtqnia: decode response: %w", err)
	}
	return nil
}

func readAPIError(resp *http.Response) *APIError {
	defer resp.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxResponseSize))
	data := struct {
		Message string `json:"message"`
		Code    string `json:"code"`
	}{}
	_ = json.Unmarshal(body, &data)
	if data.Message == "" {
		data.Message = fmt.Sprintf("API request failed with status %d", resp.StatusCode)
	}
	return &APIError{StatusCode: resp.StatusCode, RequestID: resp.Header.Get("X-Request-ID"), Code: data.Code, Message: data.Message, RetryAfter: parseRetryAfter(resp.Header.Get("Retry-After")), Body: string(body)}
}

func classifyAPIError(err *APIError) error {
	switch err.StatusCode {
	case 401:
		return &AuthenticationError{err}
	case 403:
		return &AuthorizationError{err}
	case 404:
		return &NotFoundError{err}
	case 409:
		return &ConflictError{err}
	case 429:
		return &RateLimitError{err}
	default:
		if err.StatusCode >= 500 {
			return &ServerError{err}
		}
		return err
	}
}

func retryable(status int) bool {
	return status == 429 || status == 500 || status == 502 || status == 503 || status == 504
}
func (c *Client) backoff(attempt int) time.Duration {
	wait := time.Duration(float64(c.retry.InitialWait) * math.Pow(2, float64(attempt)))
	if wait > c.retry.MaxWait {
		return c.retry.MaxWait
	}
	return wait
}
func sleep(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-timer.C:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}
func parseRetryAfter(value string) time.Duration {
	if seconds, err := strconv.Atoi(strings.TrimSpace(value)); err == nil {
		return time.Duration(max(seconds, 0)) * time.Second
	}
	if date, err := http.ParseTime(value); err == nil {
		if wait := time.Until(date); wait > 0 {
			return wait
		}
	}
	return 0
}
func isTimeout(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}
