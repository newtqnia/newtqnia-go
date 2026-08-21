package main

import (
	"context"
	"fmt"
	"log"

	newtqnia "github.com/newtqnia/newtqnia-go"
)

func main() {
	client := newtqnia.New()
	digest, err := client.News.Latest(context.Background(), &newtqnia.NewsListParams{Limit: 5})
	if err != nil {
		log.Fatal(err)
	}
	for _, article := range digest.Articles {
		fmt.Printf("%s\n%s\n\n", article.Title, article.URL)
	}
}
