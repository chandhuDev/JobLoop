package main

import (
	"context"
	"log/slog"
	"net/http"
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

	// Handle signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		slog.Info("Signal received, initiating shutdown...", slog.String("signal", sig.String()))
		cancel()
	}()

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Run the app
	exitCode := run(ctx)

	slog.Info("Shutdown complete")
	os.Exit(exitCode)
}

func run(ctx context.Context) int {
	// Error service
	errConfig := service.SetUpErrorClient()
	errInstance := &service.ErrorService{ErrorHandler: errConfig}
	go errInstance.HandleError()
	defer errInstance.Close()

	// Database
	dbInstance := dbService.ConnectDatabase()
	if dbInstance == nil {
		slog.Error("failed to connect to database")
		return 1
	}
	dbSvc := &dbService.DatabaseService{DB: dbInstance}
	defer dbSvc.Close()

	if err := dbSvc.CreateSchema(); err != nil {
		slog.Error("error creating schema", slog.Any("error", err))
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
		slog.Error("error creating browser", slog.Any("error", err))
		return 1
	}
	defer func() {
		slog.Info("Closing browser...")
		browserInstance.Close()
		slog.Info("Browser closed")
	}()

	searchInstance, err := service.CreateSearchService(ctx)
	if err != nil {
		slog.Error("error creating search service", slog.Any("error", err))
		return 1
	}
	searchConfig := service.SetUpSearch(searchInstance)
	search := &service.SearchService{Search: searchConfig}

	namesChannel := service.CreateNamesChannel(200)
	defer namesChannel.CloseNamesChan()

	visionInstance, err := service.CreateVisionInstance(ctx)
	if err != nil {
		slog.Error("error creating vision service", slog.Any("error", err))
		return 1
	}
	defer func() {
		slog.Info("Closing vision client...")
		visionInstance.Close()
		slog.Info("Vision client closed")
	}()

	visionConfig := service.SetUpVision(visionInstance, ctx, namesChannel.ReturnNamesChan())
	visionWrapper := &service.VisionWrapper{Vision: visionConfig}

	// Scraper client
	scraperClient := service.SetUpScraperClient(
		browserInstance,

		visionInstance,
		search,
		errInstance,
		// dbSvc,
		namesChannel.ReturnNamesChan(),
	)

	if scraperClient == nil || scraperClient.Search == nil || scraperClient.Browser == nil {
		slog.Error("scraper client not properly initialized")
		return 1
	}

	abcdChan := make(chan models.SeedCompanyResult, 30)

	service.ScrapeJobs(scraperClient.Browser, "http://www.checkr.com")

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

	testimonialConfig := service.NewTestimonial()
	testimonial := service.TestimonialService{Testimonial: testimonialConfig}

	httpHandler := service.NewHTTPHandlerService(dbSvc.DB)
	server := &http.Server{
		Addr:    ":8081",
		Handler: httpHandler,
	}

	g, gCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		slog.Info("Starting HTTP server", slog.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return err
		}
		return nil
	})

	g.Go(func() error {
		<-gCtx.Done()
		slog.Info("Shutting down HTTP server...")
		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer shutdownCancel()
		return server.Shutdown(shutdownCtx)
	})

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in SeedCompany", slog.Any("error", r))
			}
		}()
		seedCompany.SeedCompanyConfigs(gCtx, scraperClient)
		slog.Info("SeedCompany finished")
		return nil
	})

	g.Go(func() error {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("Panic in Testimonial", slog.Any("error", r))
				close(abcdChan)
			}
		}()
		testimonial.ScrapeTestimonial(gCtx, scraperClient,
			// seedCompany.SeedCompany.ResultChan,
			abcdChan,
			*visionWrapper)
		slog.Info("Testimonial finished")
		return nil
	})

	abcdChan <- models.SeedCompanyResult{
		CompanyName:   "precisely",
		CompanyURL:    "https://www.precisely.com/",
		SeedCompanyId: 1,
	}

	if err := g.Wait(); err != nil {
		slog.Error("Error in errgroup", slog.Any("error", err))
		return 1
	}

	slog.Info("All work completed successfully")
	return 0
}
