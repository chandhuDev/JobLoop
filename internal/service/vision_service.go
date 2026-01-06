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

func SetUpVision(vision *vision.ImageAnnotatorClient, context context.Context) *models.Vision {
	return &models.Vision{
		VisionClient: vision,
		VisionContext: context,
	}
}

func CreateVisionInstance(context context.Context) (*vision.ImageAnnotatorClient, error) {
	v, err := vision.NewImageAnnotatorClient(context)
	return v, err
}

func (v *VisionWrapper) ExtractImageFromText(ImageUrlArrays []string, errHandler interfaces.ErrorClient, w int) []models.TestimonialResult {
	slog.Info("we are starting vision scraper")
	var requests []*visionpb.AnnotateImageRequest
	for _, imageURL := range ImageUrlArrays {
		slog.Info("vision imageurl to scrape", slog.String("imageUrl", imageURL), slog.Int("of workerId", w))

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
		slog.Info("vision requesturl")

	}

	batchRequest := &visionpb.BatchAnnotateImagesRequest{
		Requests: requests,
	}
	slog.Info("before batchrequest")

	responseArray, err := v.Vision.VisionClient.BatchAnnotateImages(v.Vision.VisionContext, batchRequest)
	slog.Info("after batchrequest")

	if err != nil {
		errHandler.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "Error in vision API request:",
			Err:      err,
		})
	}
	var resultsArray []models.TestimonialResult
	for _, response := range responseArray.Responses {
		slog.Info("after batchrequest")

		slog.Info("Text Annotations: for imageurl response", response.TextAnnotations)
		resultsArray = append(resultsArray, models.TestimonialResult{
			Name: response.TextAnnotations[0].Description,
			Uri:  response.Context.Uri,
		})
	}
	return resultsArray
}
