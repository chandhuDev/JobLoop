package service

import (
	"context"
	"log/slog"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"

	vision "cloud.google.com/go/vision/apiv1"
	visionpb "cloud.google.com/go/vision/v2/apiv1/visionpb"
)

type VisionWrapper struct {
	Vision *models.Vision
}

func SetUpVision(vision *vision.ImageAnnotatorClient) *models.Vision {
	return &models.Vision{
		VisionClient: vision,
	}
}

func CreateVisionInstance(context context.Context) (*vision.ImageAnnotatorClient, error) {
	v, err := vision.NewImageAnnotatorClient(context)
	return v, err
}

func (v *VisionWrapper) ExtractImageFromText(ImageUrlArrays []string, errHandler interfaces.ErrorClient) []models.TestimonialResult {
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
	responseArray, err := v.Vision.VisionClient.BatchAnnotateImages(context.Background(), batchRequest)

	if err != nil {
		errHandler.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "Error in vision API request:",
			Err:      err,
		})
	}
	var resultsArray []models.TestimonialResult
	for _, response := range responseArray.Responses {
		slog.Info("Text Annotations:",response.TextAnnotations)
		resultsArray = append(resultsArray, models.TestimonialResult{
			Name: response.TextAnnotations[0].Description,
			Uri: response.Context.Uri,
		})
	}
	return resultsArray
}
