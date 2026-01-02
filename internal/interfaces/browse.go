package interfaces

import (
	"context"

	"github.com/chromedp/chromedp"
)

type BrowserClient interface {
	RunInNewTab(actions ...chromedp.Action) (context.Context, context.CancelFunc)
	Close()
}
