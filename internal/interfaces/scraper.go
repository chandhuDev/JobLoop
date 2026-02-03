package interfaces

import (
	// vision "cloud.google.com/go/vision/apiv1"
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
