package service

import (
	"context"
	"fmt"

	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
)

type BrowserService struct {
	Browser *models.Browser
}

func CreateNewBrowser(options models.Options, ctx context.Context) (*BrowserService, error) {
	pw, err := playwright.Run()
	if err != nil {
		return nil, err
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(options.Headless),
		Args: []string{
			"--headless=new", // Use Chrome's new headless mode
            "--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
			"--disable-setuid-sandbox",
			"--disable-infobars",
			"--window-size=1920,1080",
			"--start-maximized",
			"--disable-extensions",
			"--disable-plugins-discovery",
			"--disable-background-networking",
		},
	})
	if err != nil {
		pw.Stop()
		return nil, err
	}

	browserContext, err := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		Locale:            playwright.String("en-US"),
		TimezoneId:        playwright.String("America/New_York"),
		ColorScheme:       playwright.ColorSchemeLight,
		JavaScriptEnabled: playwright.Bool(true),
		ExtraHttpHeaders: map[string]string{
			"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"Accept-Language":           "en-US,en;q=0.9",
			"Accept-Encoding":           "gzip, deflate, br",
			"Connection":                "keep-alive",
			"Upgrade-Insecure-Requests": "1",
			"Sec-Fetch-Dest":            "document",
			"Sec-Fetch-Mode":            "navigate",
			"Sec-Fetch-Site":            "none",
			"Sec-Fetch-User":            "?1",
			"Cache-Control":             "max-age=0",
		},
	})
	if err != nil {
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("could not create context: %w", err)
	}

	// Add stealth init script to the context - runs on every new page
	err = browserContext.AddInitScript(playwright.Script{
		Content: playwright.String(`
			// Overwrite the 'webdriver' property
			Object.defineProperty(navigator, 'webdriver', {
				get: () => undefined,
			});

			// Remove webdriver from navigator
			delete navigator.__proto__.webdriver;

			// Overwrite the 'plugins' property to look real
			Object.defineProperty(navigator, 'plugins', {
				get: () => {
					const plugins = [
						{ name: 'Chrome PDF Plugin', filename: 'internal-pdf-viewer' },
						{ name: 'Chrome PDF Viewer', filename: 'mhjfbmdgcfjbbpaeojofohoefgiehjai' },
						{ name: 'Native Client', filename: 'internal-nacl-plugin' },
					];
					plugins.length = 3;
					return plugins;
				},
			});

			// Overwrite the 'languages' property
			Object.defineProperty(navigator, 'languages', {
				get: () => ['en-US', 'en'],
			});

			// Pass the Chrome check
			window.chrome = {
				runtime: {},
				loadTimes: function() {},
				csi: function() {},
				app: {},
			};

			// Pass the permissions check
			const originalQuery = window.navigator.permissions.query;
			window.navigator.permissions.query = (parameters) => (
				parameters.name === 'notifications' ?
					Promise.resolve({ state: Notification.permission }) :
					originalQuery(parameters)
			);

			// Overwrite the 'platform' property
			Object.defineProperty(navigator, 'platform', {
				get: () => 'Win32',
			});

			// Overwrite the 'hardwareConcurrency' property
			Object.defineProperty(navigator, 'hardwareConcurrency', {
				get: () => 8,
			});

			// Overwrite the 'deviceMemory' property
			Object.defineProperty(navigator, 'deviceMemory', {
				get: () => 8,
			});

			// Fix iframe contentWindow
			const originalContentWindow = Object.getOwnPropertyDescriptor(HTMLIFrameElement.prototype, 'contentWindow');
			Object.defineProperty(HTMLIFrameElement.prototype, 'contentWindow', {
				get: function() {
					const win = originalContentWindow.get.call(this);
					if (win) {
						Object.defineProperty(win.navigator, 'webdriver', { get: () => undefined });
					}
					return win;
				}
			});

			// Mock the WebGL vendor and renderer
			const getParameterProto = WebGLRenderingContext.prototype.getParameter;
			WebGLRenderingContext.prototype.getParameter = function(param) {
				if (param === 37445) return 'Intel Inc.';
				if (param === 37446) return 'Intel Iris OpenGL Engine';
				return getParameterProto.call(this, param);
			};
		`),
	})
	if err != nil {
		browserContext.Close()
		browser.Close()
		pw.Stop()
		return nil, fmt.Errorf("could not add init script: %w", err)
	}

	return &BrowserService{
		Browser: &models.Browser{
			Playwright: pw,
			Browser:    browser,
			Options:    options,
			Context:    browserContext,
		},
	}, nil
}

// RunInNewTab - USE CONTEXT.NewPage() not Browser.NewPage()
func (b *BrowserService) RunInNewTab() (playwright.Page, error) {
	// THIS IS THE KEY FIX - use Context.NewPage() to inherit all settings
	page, err := b.Browser.Context.NewPage()
	if err != nil {
		return nil, err
	}
	return page, nil
}

func (b *BrowserService) Close() {
	if b.Browser.Context != nil {
		b.Browser.Context.Close()
	}
	if b.Browser.Browser != nil {
		b.Browser.Browser.Close()
	}
	if b.Browser.Playwright != nil {
		b.Browser.Playwright.Stop()
	}
}