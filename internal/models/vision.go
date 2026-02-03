package models

import (
	// vision "cloud.google.com/go/vision/apiv1"
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

// type Vision struct {
// 	VisionClient *vision.ImageAnnotatorClient
//     VisionContext context.Context
// }

type Vision struct {
	VisionClient  *anthropic.Client
	NamesChan     *NamesClient
	VisionContext context.Context
}
