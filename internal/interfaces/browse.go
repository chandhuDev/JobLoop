package interfaces

import (
	"github.com/playwright-community/playwright-go"
)

type BrowserClient interface {
	RunInNewTab() (playwright.Page, error)
	Close()
}
