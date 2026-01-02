package models

import (
	vision "cloud.google.com/go/vision/apiv1"
)

type Vision struct {
	VisionClient *vision.ImageAnnotatorClient
	Err          ErrorHandler
}
