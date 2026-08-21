package newtqnia_test

import (
	"context"
	"fmt"
	"log"

	newtqnia "github.com/newtqnia/newtqnia-go"
)

func ExampleClient() {
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
}
