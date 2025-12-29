package error

type WorkerError struct {
   WorkerId int
   Message string
   Err error
}