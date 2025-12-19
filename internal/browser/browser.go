package browser

import (
	"fmt"
	"context"
	"sync"
	"time"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chandhuDev/JobLoop/internal/schema"
)

type Browser struct {
	allocCtx    context.Context
	allocCancel context.CancelFunc
	browserCtx  context.Context
	browserCancel context.CancelFunc
	opts Options
}

type Options struct {
	Headless bool
	Disbale_gpu bool
	WindowWidth int
    WindowHeight int
}

func CreateNewBrowser(options Options) (*Browser, err) {
    execOptions := append(chromdep.DefaultExecAllocatorOptions[:],
	                chromedp.Flag("headless", opts.Headless),
					chromedp.Flag("disable-gpu", opts.Disbale_gpu)
					chromedp.WindowSize(opts.WindowWidth, opts.WindowHeight)
    )
    allocContext, allocCancel := chromdep.NewExecAllocator(context.Background(), execpOptions...)
	browserContext, browserCancel := chromedp.NewContext(allocContext)

	if err := chromdep.Run(browserContext); err != nil {
		fmt.Printf("Error launching browser: %v\n", err)
		return nil, err
	}

	return &Browser{
		allocContext : allocContext,
		allocCancel : allocCancel,
		browserContext : browserContext,
		browserCancel : browserCancel,
		options : options
	}, nil
}

func (b *Browser) RunInNewTab(actions ...chromedp.Action) context.Context, context.CancelFunc {
	tabContext, tabCancel := chromedp.NewContext(b.browserContext)
	return tabContext, tabCancel
}

func (b *Browser) Close() {
	b.browserCancel()
	b.allocCancel()
}
