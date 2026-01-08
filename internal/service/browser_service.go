package service

import (
	"context"

	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
)

type BrowserService struct {
	Browser *models.Browser
}

func CreateNewBrowser(options models.Options, ctx context.Context) (*models.Browser, error) {
	// Start Playwright
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	// Launch browser with options
	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(options.Headless),
		Args: []string{
			"--disable-gpu",
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, err
	}

	return &models.Browser{
		Playwright: pw,
		Browser:    browser,
		Options:    options,
	}, nil
}

func (b *BrowserService) RunInNewTab() (playwright.Page, error) {
	page, err := b.Browser.Browser.NewPage(playwright.BrowserNewPageOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  b.Browser.Options.WindowWidth,
			Height: b.Browser.Options.WindowHeight,
		},
	})
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (b *BrowserService) Close() {
	if b.Browser.Browser != nil {
		b.Browser.Browser.Close()
	}
	if b.Browser.Playwright != nil {
		b.Browser.Playwright.Stop()
	}
}
