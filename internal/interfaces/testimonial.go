package interfaces

import (
	"context"

	"github.com/playwright-community/playwright-go"

	"github.com/chandhuDev/JobLoop/internal/models"
)

type TestimonialScraper interface {
	ScrapeTestimonial(ctx context.Context, scraper *ScraperClient, scChan <-chan models.SeedCompanyResult)
	scrapeCompany(ctx context.Context, page playwright.Page, scr models.SeedCompanyResult) []string
}
