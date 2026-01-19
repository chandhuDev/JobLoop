package main

import (
	"log/slog"
	"os"
	"time"

	"context"
	"os/signal"
	"syscall"

	models "github.com/chandhuDev/JobLoop/internal/models"
	"golang.org/x/sync/errgroup"

	service "github.com/chandhuDev/JobLoop/internal/service"
	"github.com/joho/godotenv"

	dbService "github.com/chandhuDev/JobLoop/internal/database"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

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

	dbInstance := dbService.ConnectDatabase()
	dbSvc := &dbService.DatabaseService{DB: dbInstance}
	err := dbSvc.CreateSchema()
	if err != nil {
		errInstance.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error in creating Database instance client",
			Err:      err,
		})
	}

	browserOptions := models.Options{
		Headless:     true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
	browserInstance, browserError := service.CreateNewBrowser(browserOptions, ctx)
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating browser instance",
		Err:      browserError,
	})
	browser := &service.BrowserService{Browser: browserInstance}

	defer func() {
		if r := recover(); r != nil {
			slog.Info("Panic recovered: %v", r)
		}
		slog.Info("Cleaning up...")
		browser.Close()
		slog.Info("Cleanup complete")
	}()

	searchInstance, searchInstanceError := service.CreateSearchService(ctx)
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating google search instance",
		Err:      searchInstanceError,
	})
	searchConfig := service.SetUpSearch(searchInstance)
	search := &service.SearchService{Search: searchConfig}

	visionInstance, visionInstanceError := service.CreateVisionInstance(ctx)
	errInstance.Send(models.WorkerError{
		WorkerId: -1,
		Message:  "error in creating google vision instance",
		Err:      visionInstanceError,
	})
	visionConfig := service.SetUpVision(visionInstance, ctx)
	visionWrapper := &service.VisionWrapper{Vision: visionConfig}

	scraperClient := service.SetUpScraperClient(browser, visionInstance, search, errInstance, dbSvc)

	SeedCompanyConfigs := []models.SeedCompany{
		{
			Name:     "Y Combinator",
			URL:      "http://www.ycombinator.com/companies",
			Selector: `a[href^="/companies/"]`,
			WaitTime: 3 * time.Second,
		},
		{
			Name:     "Peer list",
			URL:      "https://peerlist.io/jobs",
			Selector: `a[href^="/company/"][href*="/careers/"]`,
			WaitTime: 3 * time.Second,
		},
	}
	seedCompanyFirst := service.NewSeedCompanyScraper(SeedCompanyConfigs[0])
	seedCompanySecond := service.NewSeedCompanyScraper(SeedCompanyConfigs[1])

	seedCompanyArrayInstance := service.NewSeedCompanyArray(*seedCompanyFirst, *seedCompanySecond)
	seedCompany := service.SeedCompanyService{SeedCompany: seedCompanyArrayInstance}

	testimonialconfig := service.NewTestimonial()
	testimonial := service.TestimonialService{Testimonial: testimonialconfig}

	g, gCtx := errgroup.WithContext(ctx)

	go func() {
		sig := <-sigChan
		slog.Info("Signal received", "signal", sig)
		cancel()
	}()

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in SeedCompany", "error", r)
			}
		}()
		seedCompany.SeedCompanyConfigs(gCtx, scraperClient)
		return nil
	})

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in Testimonial", "error", r)
			}
		}()
		testimonial.ScrapeTestimonial(scraperClient, seedCompany.SeedCompany.ResultChan, *visionWrapper)
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("Error", "error", err)
	}

	slog.Info("All work completed")
}
