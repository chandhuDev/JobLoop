package service

import (
	"context"
	"io"
	"log/slog"
	"net/http"

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
		VisionClient:  vision,
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
	var validURLs []string

	testURL := "https://www.fieldguide.io/hs-fs/hubfs/Fieldguide/Images/ce_wa-logo-.png?width=640&height=188&name=ce_wa-logo-.png"

	abcd := []string{
		testURL,
	}
	for _, imageURL := range abcd {

		imageBytes, err := downloadImage(imageURL)
		if err != nil {
			slog.Warn("Failed to download image",
				slog.String("url", imageURL),
				slog.Any("error", err),
			)
			continue
		}

		slog.Info("bytes", slog.Any("bytes", imageBytes))

		image := &visionpb.Image{
			Content: imageBytes,
		}

		feature := &visionpb.Feature{
			Type: visionpb.Feature_TEXT_DETECTION,
		}

		req := &visionpb.AnnotateImageRequest{
			Image:    image,
			Features: []*visionpb.Feature{feature},
		}

		requests = append(requests, req)
		validURLs = append(validURLs, imageURL)
	}

	if len(requests) == 0 {
		slog.Warn("No valid images to process")
		return nil
	}

	batchReq := &visionpb.BatchAnnotateImagesRequest{
		Requests: requests,
	}

	resp, err := v.Vision.VisionClient.BatchAnnotateImages(v.Vision.VisionContext, batchReq)
	slog.Info("ee", slog.Any("dd", resp))
	if err != nil {
		errHandler.Send(models.WorkerError{
			WorkerId: w,
			Message:  "Error in vision API request",
			Err:      err,
		})
		return nil
	}

	if resp == nil || len(resp.Responses) == 0 {
		slog.Warn("No responses from vision API")
		return nil
	}

	var resultsArray []models.TestimonialResult
	for i, r := range resp.Responses {
		if r.Error != nil {
			slog.Error("Vision error", slog.String("msg", r.Error.Message))
			continue
		}

		if len(r.TextAnnotations) > 0 {
			resultsArray = append(resultsArray, models.TestimonialResult{
				Name: r.TextAnnotations[0].Description,
				Uri:  validURLs[i],
			})
		}
	}

	return resultsArray
}

func downloadImage(imageURL string) ([]byte, error) {
	req, err := http.NewRequest("GET", imageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "image/*")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}

	return io.ReadAll(resp.Body)
}
