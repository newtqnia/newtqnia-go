package main

import (
	"context"
	"errors"
	"log"

	newtqnia "github.com/newtqnia/newtqnia-go"
)

func main() {
	_, err := newtqnia.New().News.Latest(context.Background(), nil)
	if err == nil {
		return
	}
	var limited *newtqnia.RateLimitError
	var apiErr *newtqnia.APIError
	switch {
	case errors.As(err, &limited):
		log.Printf("rate limited; retry after %s", limited.RetryAfter)
	case errors.As(err, &apiErr):
		log.Printf("API error %d (%s)", apiErr.StatusCode, apiErr.RequestID)
	default:
		log.Print(err)
	}
}
