package service

import (
	"context"
	"fmt"

	vision "cloud.google.com/go/vision/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
)

type VisionConfig struct {
	visionClient *vision.ImageAnnotatorClient
}

type TestimonialResult struct {
	Name string
	Uri  string
}

func SetUpVision(vision *vision.ImageAnnotatorClient) *VisionConfig {
	return &VisionConfig{
		visionClient: vision,
	}
}

func (v VisionConfig) ExtractImageFromText(ImageUrlArrays []string) []TestimonialResult {
	var requests []*visionpb.AnnotateImageRequest
	for _, imageURL := range ImageUrlArrays {
		image := &visionpb.Image{
			Source: &visionpb.ImageSource{
				ImageUri: imageURL,
			},
		}
		feature := &visionpb.Feature{
			Type: visionpb.Feature_TEXT_DETECTION,
		}
		req := &visionpb.AnnotateImageRequest{
			Image:    image,
			Features: []*visionpb.Feature{feature},
		}
		requests = append(requests, req)
	}

	batchRequest := &visionpb.BatchAnnotateImagesRequest{
		Requests: requests,
	}
	responseArray, err := v.visionClient.BatchAnnotateImages(context.Background(), batchRequest)
	if err != nil {
		fmt.Println("Error in vision API request:", err)
	}
	var resultsArray []TestimonialResult
	for _, response := range responseArray.Responses {
		fmt.Println("Text Annotations:", response.TextAnnotations)
		resultsArray = append(resultsArray, TestimonialResult{
			Name: response.TextAnnotations[0].Description,
		})
	}
	return resultsArray
}
