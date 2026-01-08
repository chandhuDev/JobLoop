package models

import "github.com/playwright-community/playwright-go"

type Browser struct {
	Playwright *playwright.Playwright
	Browser    playwright.Browser
	Options    Options
}

type Options struct {
	Headless     bool
	WindowWidth  int
	WindowHeight int
}
