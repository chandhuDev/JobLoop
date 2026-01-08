package service

import (
	"log/slog"
	"sync"
	"time"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
)

type TestimonialService struct {
	Testimonial *models.Testimonial
}

func NewTestimonial() *models.Testimonial {
	return &models.Testimonial{
		ImageResultChan: make(chan []string, 100),
		TestimonialWg:   &sync.WaitGroup{},
		ImageWg:         &sync.WaitGroup{},
		// Remove TNodes - not needed with Playwright
	}
}

func (t *TestimonialService) ScrapeTestimonial(scraper *interfaces.ScraperClient, scChan <-chan models.SeedCompanyResult, vision VisionWrapper) {
	for i := 0; i < 3; i++ {
		t.Testimonial.TestimonialWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.TestimonialWg.Done()

			// Create a new page (equivalent to new tab)
			page, err := scraper.Browser.RunInNewTab()
			if err != nil {
				scraper.Err.Send(models.WorkerError{
					WorkerId: workerID,
					Message:  "Error creating new page",
					Err:      err,
				})
				return
			}
			defer page.Close()

			// XPath selector for testimonial images
			xpath := `xpath=//*[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'trust')]/following::*[count(.//img) >= 10][1]//img`

			slog.Info("Starting Testimonial goroutine", slog.Int("goroutine id", workerID))

			for scr := range scChan {
				slog.Info("START processing", slog.String("company", scr.CompanyName), slog.Time("time", time.Now()))

				// Navigate to the company URL
				_, err := page.Goto(scr.CompanyURL, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
					Timeout:   playwright.Float(30000),
				})
				if err != nil {
					scraper.Err.Send(models.WorkerError{
						WorkerId: workerID,
						Message:  "Error navigating to testimonial page",
						Err:      err,
					})
					continue
				}

				// Wait for body to be visible
				page.WaitForSelector("body", playwright.PageWaitForSelectorOptions{
					State:   playwright.WaitForSelectorStateVisible,
					Timeout: playwright.Float(10000),
				})

				slog.Info("END processing", slog.String("company", scr.CompanyName), slog.Time("time", time.Now()))

				// Find testimonial images using XPath
				locator := page.Locator(xpath)
				count, err := locator.Count()
				if err != nil || count == 0 {
					slog.Info("No testimonial images found for", slog.String("company Name", scr.CompanyName))
					continue
				}

				var urlArray []string

				for j := 0; j < count; j++ {
					img := locator.Nth(j)

					// Try src first, then data-src
					fullURL, _ := img.GetAttribute("src")
					if fullURL == "" || fullURL == "null" {
						fullURL, _ = img.GetAttribute("data-src")
					}

					if fullURL != "" {
						urlArray = append(urlArray, fullURL)
						slog.Info("Url extracted", slog.String("URL", fullURL), slog.Int("of workerId", workerID))
					}
				}

				if len(urlArray) > 0 {
					t.Testimonial.ImageResultChan <- urlArray
				}

				slog.Info("length of UrlArray of testimonials", slog.Int("length", len(urlArray)))
			}
		}(i)
	}

	// Image processing workers
	for i := 0; i < 2; i++ {
		t.Testimonial.ImageWg.Add(1)
		go func(workerID int) {
			defer t.Testimonial.ImageWg.Done()

			slog.Info("Started image goroutine", slog.Int("goroutine_id", workerID))

			for urlArray := range t.Testimonial.ImageResultChan {
				slog.Info("In processed Images of testimonials", slog.Int("worker id", workerID))
				vision.ExtractImageFromText(urlArray, scraper.Err, workerID)
			}
		}(i)
	}

	t.Testimonial.TestimonialWg.Wait()
	close(t.Testimonial.ImageResultChan)
	t.Testimonial.ImageWg.Wait()
}
