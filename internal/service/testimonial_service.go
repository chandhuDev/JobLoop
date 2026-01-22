package service

import (
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/repository"
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
	}
}

func (t *TestimonialService) ScrapeTestimonial(scraper *interfaces.ScraperClient, scChan <-chan models.SeedCompanyResult, vision VisionWrapper) {
	for i := 0; i < 3; i++ {
		t.Testimonial.TestimonialWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.TestimonialWg.Done()

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

			xpath := `xpath=//*[self::h1 or self::h2 or self::p or self::span or self::div][contains(translate(normalize-space(text()),'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),'trust') or contains(translate(normalize-space(text()),'ABCDEFGHIJKLMNOPQRSTUVWXYZ','abcdefghijklmnopqrstuvwxyz'),'customers')]/parent::*`

			slog.Info("Starting Testimonial goroutine", slog.Int("goroutine id", workerID))

			for scr := range scChan {
				t.Testimonial.SeedCompanyId = scr.SeedCompanyId
				slog.Info("START processing", slog.String("company", scr.CompanyName), slog.Time("time", time.Now()))
				url := scr.CompanyURL
				if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
					url = "https://" + url
				}

				_, err := page.Goto("https://www.sibros.tech/", playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateNetworkidle,
					Timeout:   playwright.Float(60000),
				})
				if err != nil {
					slog.Error("Navigation failed",
						slog.String("company", scr.CompanyName),
						slog.String("url", scr.CompanyURL),
						slog.Any("error", err))
					scraper.Err.Send(models.WorkerError{
						WorkerId: workerID,
						Message:  "Error navigating to testimonial page",
						Err:      err,
					})
					continue
				}
				slog.Info("Navigation successful", slog.String("company", scr.CompanyName))
				locator := page.Locator(xpath)

				count, err := locator.Count()
				slog.Info("counting testimonial parent nodes", slog.String("company Name", scr.CompanyName))
				slog.Info("counting testimonial parent nodes", slog.String("company Name", scr.CompanyName))
				slog.Info("counting testimonial parent nodes", slog.String("company Name", scr.CompanyName))
				slog.Info("counting testimonial parent nodes", slog.String("company Name", scr.CompanyName))
				slog.Info("counting testimonial parent nodes", slog.String("company Name", scr.CompanyName))

				slog.Info("count of testimonial nodes", slog.Int("count", count))
				if err != nil || count == 0 {
					slog.Info("No testimonial images found for", slog.String("company Name", scr.CompanyName))
					continue
				}

				minRequiredImages := 10
				maxImgCount := 0
				var bestLocator playwright.Locator

				for k := 0; k < count; k++ {
					htmlNode := locator.Nth(k)

					for level := 0; level < 2; level++ {
						var currentNode playwright.Locator

						if level == 0 {
							currentNode = htmlNode
						} else {
							parentXpath := "xpath=" + strings.Repeat("/..", level)
							currentNode = htmlNode.Locator(parentXpath)
						}

						imgLocator := currentNode.Locator("img")
						imgCount, err := imgLocator.Count()
						if err != nil {
							continue
						}

						slog.Info("Checking node of",
							slog.String("company ", scr.CompanyName),
							slog.Int("result_index", k),
							slog.Int("level", level),
							slog.Int("img_count", imgCount))

						if imgCount > maxImgCount {
							maxImgCount = imgCount
							bestLocator = imgLocator
						}

						if imgCount >= minRequiredImages {
							slog.Error("logging error and breaking first if")
							break
						}
					}

					if maxImgCount >= minRequiredImages {
						slog.Error("logging error and breaking second if")

						break
					}
				}

				if maxImgCount < minRequiredImages {
					slog.Warn("Could not find node with enough images", slog.Int("max_found", maxImgCount))
				}

				var urlArray []string
				if bestLocator != nil {
					imgCount, _ := bestLocator.Count()
					slog.Int("count of best image nodes", imgCount)
					for j := 0; j < imgCount; j++ {
						img := bestLocator.Nth(j)

						fullURL, _ := img.GetAttribute("src")
						if fullURL == "" || fullURL == "null" {
							fullURL, _ = img.GetAttribute("data-src")
						}

						if fullURL == "" || strings.HasPrefix(fullURL, "data:") {
							continue
						}

						if fullURL != "" {
							slog.Info("extarcted", slog.String("URL is", fullURL))
							urlArray = append(urlArray, fullURL)
						}
					}
				}

				if len(urlArray) > 0 {

					t.Testimonial.ImageResultChan <- urlArray
				}

			}
		}(i)
	}
	for i := 0; i < 2; i++ {
		t.Testimonial.ImageWg.Add(1)
		go func(workerID int) {
			defer t.Testimonial.ImageWg.Done()

			for urlArray := range t.Testimonial.ImageResultChan {
				slog.Info("Processing images", slog.Int("worker_id", workerID), slog.Int("url_count", len(urlArray)))
				VisionResultArray := vision.ExtractTextFromImage(urlArray, scraper.Err, workerID)
				if err := repository.BulkUpsertTestimonials(scraper.DbClient.GetDB(), t.Testimonial.SeedCompanyId, VisionResultArray); err != nil {
					slog.Error("error upserting testimonial images", slog.Any("error", err))
				}
				searchEngineKey := os.Getenv("GOOGLE_SEARCH_ENGINE")

				for _, name := range VisionResultArray {
					slog.Info("starting goroutine for peerlist search scraper", slog.Int("id", workerID))

					result, err := scraper.Search.SearchKeyWordInGoogle(
						name, workerID, searchEngineKey,
					)

					if err != nil {
						scraper.Err.Send(models.WorkerError{
							WorkerId: workerID,
							Message:  "error searching google",
							Err:      err,
						})
						continue
					}
					scr := repository.CreateSeedCompanyRepository(name, result)
					if err := repository.CreateSeedCompany(scr, scraper.DbClient.GetDB()); err != nil {
						scraper.Err.Send(models.WorkerError{
							WorkerId: workerID,
							Message:  "error creating seed company in DB",
							Err:      err,
						})
					}
				}
			}
		}(i)
	}

	t.Testimonial.TestimonialWg.Wait()
	close(t.Testimonial.ImageResultChan)
	t.Testimonial.ImageWg.Wait()
}
