package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/server/database"
	"github.com/chandhuDev/JobLoop/server/handlers"
)

func main() {
	logger.Init(logger.DefaultConfig())

	requiredEnvs := []string{"DB_USER", "DB_PASSWORD", "DB_HOST"}
	for _, env := range requiredEnvs {
		if os.Getenv(env) == "" {
			logger.Error().Str("var", env).Msg("required env variable not set")
			os.Exit(1)
		}
	}

	db, err := database.Connect()
	if err != nil {
		logger.Error().Err(err).Msg("failed to connect to database")
		os.Exit(1)
	}
	defer database.Close(db)

	logger.Info().Msg("Database connected successfully")

	h := handlers.NewHandlers(db)
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	server := &http.Server{
		Addr:    ":5001",
		Handler: mux,
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info().Str("addr", server.Addr).Msg("Starting HTTP server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error().Err(err).Msg("HTTP server error")
			os.Exit(1)
		}
	}()

	sig := <-sigChan
	logger.Info().Str("signal", sig.String()).Msg("Signal received, initiating shutdown...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("HTTP server shutdown error")
		os.Exit(1)
	}

	logger.Info().Msg("Server shutdown completed")
}
