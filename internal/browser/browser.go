package browser

import (
	"context"
	"fmt"

	"github.com/chromedp/chromedp"
)

type Browser struct {
	allocContext   context.Context
	allocCancel    context.CancelFunc
	browserContext context.Context
	browserCancel  context.CancelFunc
	options        Options
}

type Options struct {
	Headless     bool
	Disbale_gpu  bool
	WindowWidth  int
	WindowHeight int
}

func CreateNewBrowser(options Options) *Browser {
	execOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", options.Headless),
		chromedp.Flag("disable-gpu", options.Disbale_gpu),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.WindowSize(options.WindowWidth, options.WindowHeight),
	)
	allocContext, allocCancel := chromedp.NewExecAllocator(context.Background(), execOptions...)
	browserContext, browserCancel := chromedp.NewContext(allocContext)

	if err := chromedp.Run(browserContext); err != nil {
		fmt.Printf("Error launching browser: %v\n", err)
		return nil
	}

	return &Browser{
		allocContext:   allocContext,
		allocCancel:    allocCancel,
		browserContext: browserContext,
		browserCancel:  browserCancel,
		options:        options,
	}
}

func (b *Browser) RunInNewTab(actions ...chromedp.Action) (context.Context, context.CancelFunc) {
	tabContext, tabCancel := chromedp.NewContext(b.browserContext)
	return tabContext, tabCancel
}

func (b *Browser) Close() {
	b.browserCancel()
	b.allocCancel()
}
