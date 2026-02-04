package models

import (
	"context"

	"github.com/anthropics/anthropic-sdk-go"
)

type Vision struct {
	VisionClient  *anthropic.Client
	NamesChan     *NamesClient
	VisionContext context.Context
}
