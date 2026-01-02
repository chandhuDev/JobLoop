package models

import (
	"time"

	"github.com/chromedp/cdproto/cdp"

	"sync"
)

type SeedCompany struct {
	Name     string
	URL      string
	Selector string
	WaitTime time.Duration
}

type SeedCompanyResult struct {
	CompanyName string
	CompanyURL  string
}

type SeedCompanyArray struct {
	Companies  []SeedCompany
	Wg         *sync.WaitGroup
	ResultChan chan SeedCompanyResult
	Nodes      []*cdp.Node
	Err        ErrorHandler
}
