package error

import (
	"fmt"
)

func HandleError(errChan chan WorkerError){
    for err := range errChan{
		fmt.Printf("error from worker %d: message: %s\n", err.WorkerId, err.Message)
	}
}