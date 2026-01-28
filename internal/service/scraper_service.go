package service

import (
	 vision "cloud.google.com/go/vision/apiv1"
	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	 "github.com/chandhuDev/JobLoop/internal/models"
)

func SetUpScraperClient(browser interfaces.BrowserClient,
	 vision *vision.ImageAnnotatorClient, search interfaces.SearchClient, err interfaces.ErrorClient, 
	//  dbClient interfaces.DatabaseClient, 
	 namesChannel *models.NamesClient,
) *interfaces.ScraperClient {
	return &interfaces.ScraperClient{
		Browser:         browser,
		Vision:          vision,
		Search:          search,
		Err:             err,
		// DbClient:        dbClient,
		NamesChanClient: namesChannel,
	}
}
