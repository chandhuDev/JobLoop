package service

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
	"github.com/chandhuDev/JobLoop/internal/models"
)

type SearchService struct {
	Client *models.Search
}

type SearchResult struct {
	CompanyName string
	URL         string
	Error       error
}

type SearchBatchResult struct {
	CustomID string `json:"custom_id"`
	Result   struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"result"`
}

func CreateSearchService() *anthropic.Client {
	client := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)
	return &client
}

func SetUpSearch(client *anthropic.Client) *models.Search {
	return &models.Search{
		Search: client,
	}
}

func (s *SearchService) SearchKeyword(companyName string, workerId int) (string, error) {

	if len(companyName) > 30 {
		return "", fmt.Errorf("company name too long")
	}

	resp, err := s.Client.Search.Messages.New(context.TODO(), anthropic.MessageNewParams{
		Model:     anthropic.ModelClaudeSonnet4_5_20250929,
		MaxTokens: 512,
		Tools: []anthropic.ToolUnionParam{
			{
				OfTool: &anthropic.ToolParam{
					Type: "web_search_20250305",
					Name: "web_search",
				},
			},
		},
		Messages: []anthropic.MessageParam{
			{
				Role: anthropic.MessageParamRoleUser,
				Content: []anthropic.ContentBlockParamUnion{
					{
						OfText: &anthropic.TextBlockParam{
							Type: "text",
							Text: fmt.Sprintf(`Find the official website URL for the company "%s". 
Return ONLY the main domain URL (e.g., https://example.com).
If not found, return "NOT_FOUND".`, companyName),
						},
					},
				},
			},
		},
	})

	if err != nil {
		return "", err
	}

	for _, block := range resp.Content {
		if block.Type == "text" {
			url := extractURLFromText(block.Text)
			if url != "" && url != "NOT_FOUND" {
				return url, nil
			}
		}
	}

	return "", fmt.Errorf("no website found for %s", companyName)
}

func extractURLFromText(text string) string {
	text = strings.TrimSpace(text)

	words := strings.Fields(text)
	for _, word := range words {
		word = strings.TrimRight(word, ".,;:!?")
		if strings.HasPrefix(word, "http://") || strings.HasPrefix(word, "https://") {
			return word
		}
	}

	for _, word := range words {
		word = strings.TrimRight(word, ".,;:!?")
		if strings.Contains(word, ".com") || strings.Contains(word, ".io") ||
			strings.Contains(word, ".org") || strings.Contains(word, ".net") {
			if !strings.HasPrefix(word, "http") {
				return "https://" + word
			}
			return word
		}
	}

	return text
}
