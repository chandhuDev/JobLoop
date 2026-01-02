package main

import (
	"log/slog"
	"os"
	"time"

	models "github.com/chandhuDev/JobLoop/internal/models"

	"context"

	service "github.com/chandhuDev/JobLoop/internal/service"
	"github.com/joho/godotenv"
)

func main() {

	errConfig := service.SetUpErrorClient()
	errInstance := &service.ErrorService{ErrorHandler: errConfig}

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	slog.SetDefault(logger)

	if err := godotenv.Load(); err != nil {
		errInstance.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error in loading env file",
			Err:      err,
		})
	}

	browserOptions := models.Options{
		Disbale_gpu:  true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
	browserInstance, browserError := service.CreateNewBrowser(browserOptions)
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating browser instance",
		Err:      browserError,
	})
	browser := &service.BrowserService{Browser: browserInstance}
	defer browser.Close()

	searchInstance, searchInstanceError := service.CreateSearchService(context.Background())
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating google search instance",
		Err:      searchInstanceError,
	})
	searchConfig := service.SetUpSearch(searchInstance)
	search := &service.SearchService{Search: searchConfig}

	visionInstance, visionInstanceError := service.CreateVisionInstance(context.Background())
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating google vision instance",
		Err:      visionInstanceError,
	})
	visionConfig := service.SetUpVision(visionInstance)
	visionWrapper := &service.VisionWrapper{Vision: visionConfig}

	scraperClient := service.SetUpScraperClient(browser, visionInstance, search, errInstance)

	SeedCompanyConfigs := []models.SeedCompany{
		{
			Name:     "Y Combinator",
			URL:      "http://www.ycombinator.com/companies",
			Selector: `span[class^="_coName_i9oky_470"]`,
			WaitTime: 10 * time.Second,
		},
		{
			Name:     "Peer list",
			URL:      "https://peerlist.io/jobs",
			Selector: `a[href^="/company/"][href*="/careers/"]`,
			WaitTime: 5 * time.Second,
		},
	}
	seedCompanyFirst := service.NewSeedCompanyScraper(SeedCompanyConfigs[0])
	seedCompanySecond := service.NewSeedCompanyScraper(SeedCompanyConfigs[1])
	seedCompanyArrayInstance := service.NewSeedCompanyArray(*seedCompanyFirst, *seedCompanySecond)
	seedCompany := service.SeedCompanyService{SeedCompany: seedCompanyArrayInstance}

	seedCompany.SeedCompanyConfigs(scraperClient)

	testimonialconfig := service.NewTestimonial()
	testimonial := service.TestimonialService{Testimonial: testimonialconfig}
	testimonial.ScrapeTestimonial(scraperClient, seedCompany.SeedCompany.ResultChan, *visionWrapper)
}
