package interfaces

import "github.com/chandhuDev/JobLoop/internal/models"

type ErrorClient interface {
	HandleError()
	Send(e models.WorkerError)
}
