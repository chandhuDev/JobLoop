package interfaces

import (
	models "github.com/chandhuDev/JobLoop/internal/models"
)

type SeedCompanyScraper interface {
	GetSeedCompaniesFromPeerList(scraper *ScraperClient, companyConfig models.SeedCompany)
	GetSeedCompaniesFromYCombinator(scraper *ScraperClient, companyConfig models.SeedCompany)
	SeedCompanyConfigs(scraper *ScraperClient)
	UploadSeedCompanyToChannel(scraper *ScraperClient)
}
