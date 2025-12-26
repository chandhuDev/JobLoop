package search

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

func CreateSearchService(context context.Context) *customsearch.Service {
	apiKey := os.Getenv("GOOGLE_API_KEY")

	customsearchService, err := customsearch.NewService(context, option.WithAPIKey(apiKey))
	if err != nil {
		fmt.Println("Error creating custom search service:", err)
	}

	return customsearchService
}
