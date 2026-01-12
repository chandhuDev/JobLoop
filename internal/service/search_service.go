package service

import (
	"context"
	"log/slog"
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
	// slog.Info("search scraper url anme", slog.String("url name", name), slog.Int("from worker id", i))
	v, err := s.Search.SearchClient.Cse.List().Q(name).Cx(key).Do()
	slog.Info("search results",
		slog.Any("url", v.Items[0].DisplayLink),
	)
	return v.Items[0].DisplayLink, err
}
