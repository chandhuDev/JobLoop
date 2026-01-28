package models

import (
	playwright "github.com/playwright-community/playwright-go")

type Browser struct {
	Playwright *playwright.Playwright
	Browser    playwright.Browser
	Options    Options
	Context    playwright.BrowserContext
}

type Options struct {
	Headless     bool
	WindowWidth  int
	WindowHeight int
}
