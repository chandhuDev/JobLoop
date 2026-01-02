package models

type WorkerError struct {
	WorkerId int
	Message  string
	Err      error
}

type ErrorHandler struct {
	ErrChan chan WorkerError
}
