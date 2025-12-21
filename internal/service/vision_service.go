package service

import (
	"fmt"
	vision "cloud.google.com/go/vision/apiv1"
)

type VisionConfig struct {
	vision vision.ImageAnnotatorClient
}

func SetUpVision(vision vision.ImageAnnotatorClient) *VisionConfig {
	return &VisionConfig{
		vision: vision,
	}
} 

func(v *VisionConfig) ExtractImageFromText(ImageUrl string) string {

}