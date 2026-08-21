# NewTqnia Go SDK

[![CI](https://github.com/newtqnia/newtqnia-go/actions/workflows/ci.yml/badge.svg)](https://github.com/newtqnia/newtqnia-go/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/newtqnia/newtqnia-go.svg)](https://pkg.go.dev/github.com/newtqnia/newtqnia-go)
[![Go](https://img.shields.io/badge/Go-1.22%2B-00ADD8?logo=go&logoColor=white)](go.mod)
[![License](https://img.shields.io/badge/license-MIT-blue)](LICENSE)

The official, dependency-free Go SDK for the [NewTqnia technology news API](https://newtqnia.com/en/developers). It provides typed English and Arabic news, `context.Context` cancellation, request timeouts, automatic retries, `Retry-After` support, typed errors, and configurable HTTP transports.

> When displaying API content, preserve returned article URLs and visibly render the attribution supplied in the response—normally [Powered by NewTqnia](https://newtqnia.com/en).

## Requirements and installation

Go 1.22 or newer:

```bash
go get github.com/newtqnia/newtqnia-go@latest
```

## Quick start

```go
package main

import (
    "context"
    "fmt"
    "log"

    newtqnia "github.com/newtqnia/newtqnia-go"
)

func main() {
    client := newtqnia.New()
    digest, err := client.News.Latest(context.Background(), &newtqnia.NewsListParams{
        Locale: newtqnia.LocaleEnglish,
        Limit:  5,
    })
    if err != nil {
        log.Fatal(err)
    }

    for _, article := range digest.Articles {
        fmt.Println(article.Title, article.URL)
    }
    fmt.Println(digest.Attribution.Text, digest.Attribution.URL)
}
```

The API currently exposes only the two collections in NewTqnia's [OpenAPI contract](openapi/openapi.yaml):

```go
today, err := client.News.Today(ctx, &newtqnia.NewsListParams{Locale: newtqnia.LocaleArabic})
latest, err := client.News.Latest(ctx, &newtqnia.NewsListParams{Category: "ai", Limit: 10})
```

Free-text search, individual article lookup, category listing, and pagination are not currently part of the public API, so the SDK does not invent those operations.

## Authentication and identification

No API key is required. If you have an optional key from your [NewTqnia profile](https://newtqnia.com/en/account), configure it along with optional application identification:

```go
client, err := newtqnia.NewClient(newtqnia.Config{
    APIKey:      os.Getenv("NEWTQNIA_API_KEY"),
    Application: "editorial-dashboard",
    Website:     "https://example.com",
})
```

The key is sent as `X-API-Key`; it is never placed in the URL or an `Authorization` header.

## Configuration

```go
client, err := newtqnia.NewClient(newtqnia.Config{
    BaseURL:   "https://api.newtqnia.com",
    Timeout:   15 * time.Second,
    UserAgent: "my-service/2.0",
    Retry: newtqnia.RetryConfig{
        MaxRetries:  3,
        InitialWait: 500 * time.Millisecond,
        MaxWait:     8 * time.Second,
    },
    HTTPClient: &http.Client{Transport: customTransport},
    Headers:    http.Header{"X-Trace-Source": {"my-service"}},
})
```

The default is a 30-second timeout and two retries after the initial request. Retries use exponential backoff for temporary network failures and HTTP 429, 500, 502, 503, and 504 responses. Server `Retry-After` instructions are honored. Set `DisableRetries: true` when the initial request must be the only attempt.

`Client` and `NewsService` are safe for concurrent use. Reuse a client instead of constructing one per request.

## Cancellation

```go
ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
defer cancel()

digest, err := client.News.Latest(ctx, nil)
```

Caller cancellation is returned as `context.Canceled` or `context.DeadlineExceeded`. The SDK's configured per-attempt timeout returns `*newtqnia.TimeoutError`.

## Error handling

```go
digest, err := client.News.Latest(ctx, nil)
if err != nil {
    var limited *newtqnia.RateLimitError
    var apiErr *newtqnia.APIError

    switch {
    case errors.As(err, &limited):
        log.Printf("retry after %s", limited.RetryAfter)
    case errors.As(err, &apiErr):
        log.Printf("status=%d request=%s code=%s", apiErr.StatusCode, apiErr.RequestID, apiErr.Code)
    default:
        log.Print(err)
    }
}
```

Specialized response errors are `AuthenticationError`, `AuthorizationError`, `NotFoundError`, `ConflictError`, `RateLimitError`, and `ServerError`. Configuration and parameter problems use `ValidationError`; exhausted transport failures use `NetworkError`.

## Release and Go module publishing

Go modules are distributed from Git tags rather than uploaded to a package registry. For the first public release:

```bash
git tag -a v1.0.0 -m "NewTqnia Go SDK v1.0.0"
git push origin main v1.0.0
GOPROXY=https://proxy.golang.org go list -m github.com/newtqnia/newtqnia-go@v1.0.0
```

The module path, repository URL, package name, semantic version, license, documentation, and CI workflow are ready for discovery by the Go proxy and [pkg.go.dev](https://pkg.go.dev/).

## Development

```bash
go fmt ./...
go vet ./...
go test -race -cover ./...
```

See [CONTRIBUTING.md](CONTRIBUTING.md), [SECURITY.md](SECURITY.md), the [changelog](CHANGELOG.md), and the [MIT license](LICENSE).

## Related resources

- [NewTqnia technology news](https://newtqnia.com/en)
- [Developer API documentation](https://newtqnia.com/en/developers)
- [Embeddable news widget](https://newtqnia.com/en/widget)
- [RSS and JSON feeds](https://newtqnia.com/en/feeds)
