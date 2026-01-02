package interfaces

import (
	models "github.com/chandhuDev/JobLoop/internal/models"
)

type VisionClient interface {
	ExtractImageFromText(ImageUrlArrays []string) []models.TestimonialResult
}
