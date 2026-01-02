package service

import (
	"context"
	"os"

	models "github.com/chandhuDev/JobLoop/internal/models"
	"google.golang.org/api/customsearch/v1"
	"google.golang.org/api/option"
)

type SearchService struct {
	Search *models.Search
}

func SetUpSearch(search *customsearch.Service) *models.Search {
	return &models.Search{
		SearchClient: search,
	}
}

func CreateSearchService(context context.Context) (*customsearch.Service, error) {
	apiKey := os.Getenv("GOOGLE_API_KEY")

	customsearchService, err := customsearch.NewService(context, option.WithAPIKey(apiKey))

	return customsearchService, err
}

func (s *SearchService) SearchKeyWordInGoogle(name string, i int, key string) (string, error) {
	v, err := s.Search.SearchClient.Cse.List().Q(name).Cx(key).Do()

	return v.Items[0].Link, err
}
