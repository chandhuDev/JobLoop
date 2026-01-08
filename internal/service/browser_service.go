package service

import (
	"context"
	"time"

	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chromedp/chromedp"
)

type BrowserService struct {
	Browser *models.Browser
}

func CreateNewBrowser(options models.Options, ctx context.Context) (*models.Browser, error) {
	execOptions := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", "new"),
		chromedp.Flag("disable-gpu", options.Disbale_gpu),
		chromedp.UserAgent("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		chromedp.Flag("disable-blink-features", "AutomationControlled"),
		chromedp.Flag("disable-dev-shm-usage", true),
        chromedp.Flag("no-sandbox", true),
        chromedp.Flag("disable-setuid-sandbox", true),
		chromedp.WindowSize(options.WindowWidth, options.WindowHeight),
	)
	allocContext, allocCancel := chromedp.NewExecAllocator(ctx, execOptions...)
	browserContext, browserCancel := chromedp.NewContext(allocContext)

	err := chromedp.Run(browserContext)

	return &models.Browser{
		AllocContext:   allocContext,
		AllocCancel:    allocCancel,
		BrowserContext: browserContext,
		BrowserCancel:  browserCancel,
		Options:        options,
	}, err
}

func (b *BrowserService) RunInNewTab(actions ...chromedp.Action) (context.Context, context.CancelFunc) {
	tabContext, tabCancel := chromedp.NewContext(b.Browser.BrowserContext)
	timeoutCtx, timeoutCancel := context.WithTimeout(tabContext, 120*time.Second)

	return timeoutCtx, func() {
		timeoutCancel()
		tabCancel()
	}
}

func (b *BrowserService) Close() {
	b.Browser.BrowserCancel()
	b.Browser.AllocCancel()
}
