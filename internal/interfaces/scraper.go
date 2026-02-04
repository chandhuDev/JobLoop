package interfaces

import (
	"github.com/anthropics/anthropic-sdk-go"
	"github.com/chandhuDev/JobLoop/internal/models"
)

type ScraperClient struct {
	Browser         BrowserClient
	Search          SearchClient
	Vision          *anthropic.Client
	DbClient        DatabaseClient
	NamesChanClient *models.NamesClient
}
