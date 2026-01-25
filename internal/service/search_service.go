package service

import (
	"context"
	"fmt"
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
	if s.Search == nil || s.Search.SearchClient == nil {
		return "", fmt.Errorf("search client not initialized")
	}

	if key == "" {
		return "", fmt.Errorf("search engine key is empty")
	}

	v, err := s.Search.SearchClient.Cse.List().Q(name).Cx(key).Do()
	if err != nil {
		slog.Error("search API error", slog.Any("error", err), slog.String("name", name))
		return "", err
	}

	if v == nil || len(v.Items) == 0 {
		slog.Warn("no search results found", slog.String("name", name))
		return "", fmt.Errorf("no results found for %s", name)
	}

	url := v.Items[0].DisplayLink
	slog.Info("search results", slog.String("url", url), slog.Int("workerId", i))

	return url, nil
}
