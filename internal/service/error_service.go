package service

import (
	"log/slog"

	models "github.com/chandhuDev/JobLoop/internal/models"
)

type ErrorService struct {
	ErrorHandler *models.ErrorHandler
}

func SetUpErrorClient() *models.ErrorHandler {
	return &models.ErrorHandler{
		ErrChan: make(chan models.WorkerError, 100),
	}
}
func (e *ErrorService) HandleError() {
	for err := range e.ErrorHandler.ErrChan {
		slog.Info("error from worker %d: message: %s\n", err.WorkerId, err.Message)
	}
}

func (e *ErrorService) Send(error models.WorkerError) {
	e.ErrorHandler.ErrChan <- error
}
