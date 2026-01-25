package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	dbService "github.com/chandhuDev/JobLoop/internal/database"
	models "github.com/chandhuDev/JobLoop/internal/models"
	service "github.com/chandhuDev/JobLoop/internal/service"
	"github.com/joho/godotenv"
	"golang.org/x/sync/errgroup"
)

func main() {
	if err := godotenv.Load(); err != nil {
		slog.Error("error loading env file", slog.Any("error", err))
		os.Exit(1)
	}

	requiredEnvs := []string{"GOOGLE_API_KEY", "GOOGLE_SEARCH_ENGINE"}
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			slog.Error("required env variable not set", slog.String("var", env))
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	errConfig := service.SetUpErrorClient()
	errInstance := &service.ErrorService{ErrorHandler: errConfig}

	dbInstance := dbService.ConnectDatabase()
	if dbInstance == nil {
		slog.Error("failed to connect to database")
		os.Exit(1)
	}
	dbSvc := &dbService.DatabaseService{DB: dbInstance}
	if err := dbSvc.CreateSchema(); err != nil {
		slog.Error("error creating schema", slog.Any("error", err))
		os.Exit(1)
	}

	browserOptions := models.Options{
		Headless:     true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
	browserInstance, err := service.CreateNewBrowser(browserOptions, ctx)
	if err != nil {
		slog.Error("error creating browser", slog.Any("error", err))
		os.Exit(1)
	}
	defer browserInstance.Close()

	searchInstance, err := service.CreateSearchService(ctx)
	if err != nil {
		slog.Error("error creating search service", slog.Any("error", err))
		os.Exit(1)
	}
	searchConfig := service.SetUpSearch(searchInstance)
	search := &service.SearchService{Search: searchConfig}

	namesChannel := service.CreateNamesChannel(200)

	visionInstance, err := service.CreateVisionInstance(ctx)
	if err != nil {
		slog.Error("error creating vision service", slog.Any("error", err))
		os.Exit(1)
	}
	visionConfig := service.SetUpVision(visionInstance, ctx, namesChannel.ReturnNamesChan())
	visionWrapper := &service.VisionWrapper{Vision: visionConfig}

	scraperClient := service.SetUpScraperClient(
		browserInstance,
		visionInstance,
		search,
		errInstance,
		dbSvc,
		namesChannel.ReturnNamesChan(),
	)

	if scraperClient == nil || scraperClient.Search == nil || scraperClient.Browser == nil {
		slog.Error("scraper client not properly initialized")
		os.Exit(1)
	}

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
		select {
		case sig := <-sigChan:
			slog.Info("Signal received, shutting down", slog.String("signal", sig.String()))
			cancel()
		case <-gCtx.Done():
			return
		}
	}()

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in SeedCompany", slog.Any("error", r))
			}
		}()
		seedCompany.SeedCompanyConfigs(gCtx, scraperClient)
		return nil
	})

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in Testimonial", slog.Any("error", r))
			}
		}()
		testimonial.ScrapeTestimonial(scraperClient, seedCompany.SeedCompany.ResultChan, *visionWrapper)
		return nil
	})

	if err := g.Wait(); err != nil {
		slog.Error("Error in errgroup", slog.Any("error", err))
	}

	slog.Info("All work completed")
}
