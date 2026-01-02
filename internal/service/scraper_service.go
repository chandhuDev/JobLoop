package service

import (
	vision "cloud.google.com/go/vision/apiv1"
	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
)

func SetUpScraperClient(browser interfaces.BrowserClient, vision *vision.ImageAnnotatorClient, search interfaces.SearchClient, err interfaces.ErrorClient) *interfaces.ScraperClient {
	return &interfaces.ScraperClient{
		Browser: browser,
		Vision:  vision,
		Search:  search,
		Err:     err,
	}
}
