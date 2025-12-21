package vision

import (
	"fmt"
	"context"
	vision "cloud.google.com/go/vision/apiv1"
)

func CreateVisionInit(context context.Context) (*vision.ImageAnnotatorClient, error) {
	v, err := vision.NewImageAnnotatorClient(context)
	if err != nil {
		fmt.Println("Failed to create vision client:", err)
		return nil, err
	}
	return v, err
}