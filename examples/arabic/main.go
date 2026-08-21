package main

import (
	"context"
	"fmt"
	"log"

	newtqnia "github.com/newtqnia/newtqnia-go"
)

func main() {
	digest, err := newtqnia.New().News.Today(context.Background(), &newtqnia.NewsListParams{Locale: newtqnia.LocaleArabic, Limit: 5})
	if err != nil {
		log.Fatal(err)
	}
	for _, article := range digest.Articles {
		fmt.Println(article.Title, article.URL)
	}
}
