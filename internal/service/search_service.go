package service

import (
	"context"
	"fmt"
	"os"

	"github.com/chandhuDev/JobLoop/internal/logger"
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

	v, err := s.Search.SearchClient.Cse.List().Q(name + "company official website").Cx(key).Do()
	if err != nil {
		logger.Error().Err(err).Str("name", name).Msg("search API error")
		return "", err
	}

	if v == nil || len(v.Items) == 0 {
		logger.Warn().Str("name", name).Msg("no search results found")
		return "", fmt.Errorf("no results found for %s", name)
	}

	url := v.Items[0].DisplayLink
	logger.Info().Str("url", url).Int("workerId", i).Msg("search results")

	return url, nil
}
