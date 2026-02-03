package service

import (
	// vision "cloud.google.com/go/vision/apiv1"
	"github.com/anthropics/anthropic-sdk-go"
	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/models"
)

func SetUpScraperClient(browser interfaces.BrowserClient,
	vision *anthropic.Client,
	search interfaces.SearchClient,
	dbClient interfaces.DatabaseClient,
	namesChannel *models.NamesClient,
) *interfaces.ScraperClient {
	return &interfaces.ScraperClient{
		Browser:         browser,
		Vision:          vision,
		Search:          search,
		DbClient:        dbClient,
		NamesChanClient: namesChannel,
	}
}
