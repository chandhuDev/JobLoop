package models

import (
	vision "cloud.google.com/go/vision/apiv1"
	"context"
)

type Vision struct {
	VisionClient *vision.ImageAnnotatorClient
    VisionContext context.Context
	NamesChan *NamesClient
}
