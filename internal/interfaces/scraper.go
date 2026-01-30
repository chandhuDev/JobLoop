package interfaces

import (
	vision "cloud.google.com/go/vision/apiv1"
	"github.com/chandhuDev/JobLoop/internal/models"
)

type ScraperClient struct {
	Browser         BrowserClient
	Search          SearchClient
	Vision          *vision.ImageAnnotatorClient
	DbClient        DatabaseClient
	NamesChanClient *models.NamesClient
}
