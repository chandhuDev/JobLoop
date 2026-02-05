package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	dbService "github.com/chandhuDev/JobLoop/internal/database"
	"github.com/chandhuDev/JobLoop/internal/logger"
	models "github.com/chandhuDev/JobLoop/internal/models"
	service "github.com/chandhuDev/JobLoop/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	// Initialize logger
	logger.Init(logger.DefaultConfig())

	_ = godotenv.Load()

	requiredEnvs := []string{"MAX_LEN", "ANTHROPIC_API_KEY", "DB_USER", "DB_PASSWORD", "DB_HOST"}
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			logger.Error().Str("var", env).Msg("required env variable not set")
			os.Exit(1)
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info().Str("signal", sig.String()).Msg("Signal received, initiating shutdown...")
		cancel()
	}()

	// Run the app
	exitCode := run(ctx)

	logger.Info().Msg("Shutdown complete")
	os.Exit(exitCode)
}

func run(ctx context.Context) int {

	// Database
	dbInstance := dbService.ConnectDatabase()
	if dbInstance == nil {
		logger.Error().Msg("failed to connect to database")
		return 1
	}
	dbSvc := &dbService.DatabaseService{DB: dbInstance}
	defer dbSvc.Close()

	if err := dbSvc.CreateSchema(); err != nil {
		logger.Error().Err(err).Msg("error creating schema")
		return 1
	}

	// Browser
	browserOptions := models.Options{
		Headless:     true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
	browserInstance, err := service.CreateNewBrowser(browserOptions, ctx)
	if err != nil {
		logger.Error().Err(err).Msg("error creating browser")
		return 1
	}
	defer func() {
		logger.Info().Msg("Closing browser...")
		browserInstance.Close()
		logger.Info().Msg("Browser closed")
	}()

	searchInstance := service.CreateSearchService()

	searchConfig := service.SetUpSearch(searchInstance)
	search := &service.SearchService{Client: searchConfig}

	namesChannel := service.CreateNamesChannel(200)
	defer namesChannel.CloseNamesChan()

	visionInstance := service.CreateVisionInstance()

	visionConfig := service.SetUpVision(visionInstance, ctx, namesChannel.ReturnNamesChan())
	visionWrapper := &service.VisionWrapper{Vision: visionConfig}

	// Scraper client
	scraperClient := service.SetUpScraperClient(
		browserInstance,
		visionInstance,
		search,
		dbSvc,
		namesChannel.ReturnNamesChan(),
	)

	if scraperClient == nil || scraperClient.Search == nil || scraperClient.Browser == nil {
		logger.Error().Msg("scraper client not properly initialized")
		return 1
	}

	SeedCompanyConfigs := []models.SeedCompany{
		{
			Name:     "Y Combinator",
			URL:      "http://www.ycombinator.com/companies",
			Selector: `a[href^="/companies/"]`,
			WaitTime: 10 * time.Second,
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

	testimonialConfig := service.NewTestimonial()
	testimonial := service.TestimonialService{Testimonial: testimonialConfig}

	// Channel to track scraper completion
	done := make(chan struct{})

	// Run scrapers
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("error", r).Msg("Panic in SeedCompany")
			}
		}()
		seedCompany.SeedCompanyConfigs(ctx, scraperClient)
		logger.Info().Msg("SeedCompany scraping completed")
	}()

	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error().Interface("error", r).Msg("Panic in Testimonial")
			}
		}()
		testimonial.ScrapeTestimonial(ctx, scraperClient,
			seedCompany.SeedCompany.ResultChan,
			*visionWrapper)
		logger.Info().Msg("Testimonial scraping completed")
		close(done)
	}()

	// Wait for either completion or cancellation
	select {
	case <-done:
		logger.Info().Msg("All scraping tasks completed successfully")
	case <-ctx.Done():
		logger.Info().Msg("Scraping interrupted by signal")
	}

	logger.Info().Msg("Shutdown complete")
	return 0
}
