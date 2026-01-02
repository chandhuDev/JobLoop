package models

import "context"

type Browser struct {
	AllocContext   context.Context
	AllocCancel    context.CancelFunc
	BrowserContext context.Context
	BrowserCancel  context.CancelFunc
	Options        Options
}

type Options struct {
	Headless     bool
	Disbale_gpu  bool
	WindowWidth  int
	WindowHeight int
}
