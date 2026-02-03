package service

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/logger"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/repository"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

var fileMutex sync.Mutex

type VisionWrapper struct {
	Vision *models.Vision
}

type OCRResult struct {
	ImageURL string
	Text     string
	Error    error
}

type BatchResult struct {
	CustomID string `json:"custom_id"`
	Result   struct {
		Type    string `json:"type"`
		Message struct {
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"message"`
		Error struct {
			Type    string `json:"type"`
			Message string `json:"message"`
		} `json:"error"`
	} `json:"result"`
}

func SetUpVision(vision *anthropic.Client, context context.Context, namesChannel *models.NamesClient) *models.Vision {
	return &models.Vision{
		VisionClient:  vision,
		VisionContext: context,
		NamesChan:     namesChannel,
	}
}

func CreateVisionInstance() *anthropic.Client {
	vClient := anthropic.NewClient(
		option.WithAPIKey(os.Getenv("ANTHROPIC_API_KEY")),
	)
	return &vClient
}

func (v *VisionWrapper) ExtractTextFromImage(
	imageURLs []string,
	scraper *interfaces.ScraperClient,
	workerID int,
	seedCompanyId uint,
) {
	if len(imageURLs) == 0 {
		logger.Info().Int("worker_id", workerID).Msg("no images to process")
		return
	}

	logger.Info().Int("worker_id", workerID).Int("image_count", len(imageURLs)).Uint("seed_company_id", seedCompanyId).Msg("starting vision scraper")

	requests, urlMap, err := createOCRRequests(imageURLs, scraper.Browser)
	if err != nil {
		logger.Error().Err(err).Msg("error creating OCR requests")
		return
	}

	if len(requests) == 0 {
		logger.Warn().Int("worker_id", workerID).Msg("no valid requests created")
		return
	}

	logger.Info().Int("request_count", len(requests)).Msg("submitting images for OCR")

	messageBatch, err := v.Vision.VisionClient.Messages.Batches.New(context.TODO(), anthropic.MessageBatchNewParams{
		Requests: requests,
	})
	if err != nil {
		logger.Error().Err(err).Msg("error submitting batch")
		return
	}

	logger.Info().Str("batch_id", messageBatch.ID).Msg("created message batch for OCR")

	pollBatch(v.Vision.VisionClient, messageBatch.ID)

	results, err := getResults(messageBatch.ID, urlMap)
	if err != nil {
		logger.Error().Err(err).Msg("error getting results")
		return
	}

	var testimonials []string
	for _, result := range results {
		if result.Error != nil {
			logger.Warn().Str("url", result.ImageURL).Err(result.Error).Msg("OCR failed for image")
			continue
		}
		if len(result.Text) > 25 {
			logger.Warn().Str("url", result.ImageURL).Int("text_length", len(result.Text)).Msg("extracted text is too long likely not just company names")
			continue
		}
		scraper.NamesChanClient.NamesChan <- result.Text

		testimonials = append(testimonials, result.Text)
	}

	if len(testimonials) > 0 {
		if err := repository.BulkUpsertTestimonials(scraper.DbClient.GetDB(), seedCompanyId, testimonials); err != nil {
			logger.Error().Err(err).Msg("error upserting testimonial images")
			return
		}
	}

	logger.Info().Int("worker_id", workerID).Int("results", len(testimonials)).Uint("seed_company_id", seedCompanyId).Msg("vision processing completed")
}

func createOCRRequests(imageURLs []string, browser interfaces.BrowserClient) ([]anthropic.MessageBatchNewParamsRequest, map[string]string, error) {
	var requests []anthropic.MessageBatchNewParamsRequest
	urlMap := make(map[string]string)

	for i, url := range imageURLs {
		customID := fmt.Sprintf("ocr-%d", i)
		urlMap[customID] = url

		ext := getExtFromURL(url)

		if ext == ".svg" {
			imageBytes, err := convertSVGtoPNG(browser, url)
			if err != nil {
				logger.Warn().Str("url", url).Err(err).Msg("failed to convert SVG, skipping")
				delete(urlMap, customID)
				continue
			}

			base64Image := base64.StdEncoding.EncodeToString(imageBytes)
			logger.Info().Str("url", url).Msg("converted SVG to PNG")

			requests = append(requests, anthropic.MessageBatchNewParamsRequest{
				CustomID: customID,
				Params: anthropic.MessageBatchNewParamsRequestParams{
					MaxTokens: 4096,
					Model:     anthropic.ModelClaudeSonnet4_5_20250929,
					Messages: []anthropic.MessageParam{
						{
							Role: anthropic.MessageParamRoleUser,
							Content: []anthropic.ContentBlockParamUnion{
								{
									OfImage: &anthropic.ImageBlockParam{
										Type: "image",
										Source: anthropic.ImageBlockParamSourceUnion{
											OfBase64: &anthropic.Base64ImageSourceParam{
												Type:      "base64",
												MediaType: "image/png",
												Data:      base64Image,
											},
										},
									},
								},
								{
									OfText: &anthropic.TextBlockParam{
										Type: "text",
										Text: "Extract all company names from this image. Return only the company names, one per line.",
									},
								},
							},
						},
					},
				},
			})
		} else {
			requests = append(requests, anthropic.MessageBatchNewParamsRequest{
				CustomID: customID,
				Params: anthropic.MessageBatchNewParamsRequestParams{
					MaxTokens: 4096,
					Model:     anthropic.ModelClaudeSonnet4_5_20250929,
					Messages: []anthropic.MessageParam{
						{
							Role: anthropic.MessageParamRoleUser,
							Content: []anthropic.ContentBlockParamUnion{
								{
									OfImage: &anthropic.ImageBlockParam{
										Type: "image",
										Source: anthropic.ImageBlockParamSourceUnion{
											OfURL: &anthropic.URLImageSourceParam{
												Type: "url",
												URL:  url,
											},
										},
									},
								},
								{
									OfText: &anthropic.TextBlockParam{
										Type: "text",
										Text: "Extract all company names from this image. Return only the company names, one per line.",
									},
								},
							},
						},
					},
				},
			})
		}
	}

	return requests, urlMap, nil
}

func convertSVGtoPNG(browser interfaces.BrowserClient, svgURL string) ([]byte, error) {
	page, err := browser.RunInNewTab()
	if err != nil {
		return nil, err
	}
	defer page.Close()

	if _, err := page.Goto(svgURL); err != nil {
		return nil, err
	}

	return page.Screenshot(playwright.PageScreenshotOptions{
		Type: playwright.ScreenshotTypePng,
	})
}

func pollBatch(client *anthropic.Client, batchID string) {
	logger.Info().Str("batch_id", batchID).Msg("polling batch status")

	for {
		batch, err := client.Messages.Batches.Get(context.TODO(), batchID)
		if err != nil {
			logger.Error().Err(err).Msg("error polling batch")
			time.Sleep(5 * time.Second)
			continue
		}

		total := batch.RequestCounts.Succeeded + batch.RequestCounts.Errored + batch.RequestCounts.Processing
		completed := batch.RequestCounts.Succeeded + batch.RequestCounts.Errored

		logger.Info().
			Str("status", string(batch.ProcessingStatus)).
			Int64("completed", completed).
			Int64("total", total).
			Msg("batch progress")

		if batch.ProcessingStatus == "ended" {
			logger.Info().Str("batch_id", batchID).Msg("batch processing ended")
			return
		}

		time.Sleep(5 * time.Second)
	}
}

func getResults(batchID string, urlMap map[string]string) ([]OCRResult, error) {
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("ANTHROPIC_API_KEY not set")
	}

	url := fmt.Sprintf("https://api.anthropic.com/v1/messages/batches/%s/results", batchID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	var results []OCRResult

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 2*1024*1024)

	for scanner.Scan() {
		var batchResult BatchResult
		if err := json.Unmarshal(scanner.Bytes(), &batchResult); err != nil {
			continue
		}

		originalURL, exists := urlMap[batchResult.CustomID]
		if !exists {
			continue
		}

		if batchResult.Result.Type == "succeeded" {
			var text string
			for _, block := range batchResult.Result.Message.Content {
				if block.Type == "text" {
					text = block.Text
					break
				}
			}
			results = append(results, OCRResult{
				ImageURL: originalURL,
				Text:     text,
			})
		} else {
			results = append(results, OCRResult{
				ImageURL: originalURL,
				Error:    fmt.Errorf("%s: %s", batchResult.Result.Error.Type, batchResult.Result.Error.Message),
			})
		}
	}

	return results, scanner.Err()
}

func getExtFromURL(url string) string {
	if idx := strings.Index(url, "?"); idx != -1 {
		url = url[:idx]
	}
	return strings.ToLower(filepath.Ext(url))
}
