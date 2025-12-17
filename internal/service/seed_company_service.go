package service

import (
	"fmt"
	"sync"
	"time"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
)

type SeedCompanyConfig struct {
	Name     string
	URL      string
	Selector string
	WaitTime time.Duration
}

type SeedCompanyResult struct {
	CompanyName string
	CompanyURL string
}

func NewSeedCompanyScraper(companyConfig SeedCompanyConfig) *SeedCompanyConfig {
	return &SeedCompanyConfig{companyConfig.Name, companyConfig.URL, companyConfig.Selector, companyConfig.WaitTime}
}

func (sj *SeedCompanyConfig) ScrapeSeedCompanies() error {
	var wg sync.WaitGroup
	wg.Add(1)
	defer wg.Done()
	err := sj.scrapeAndStoreSeedCompanies(sj)
}

func (sj *SeedCompanyConfig) scrapeAndStoreSeedCompanies() ([]SeedCompanyResult ,error) {
	var nodes []*cdp.Node
    var results []SeedCompanyResult

	fmt.Println("Scraping seed companies for:", sj.Name)

    for i := 0; i < len(nodes); i++ {
		var name string
		chromedp.Run(ctx,
			chromedp.Text(nodes[i].FullXPath(), &name, chromedp.NodeVisible),
		)

		chromedp.Run(ctx,
			chromedp.Click(nodes[i].FullXPath()),
			chromedp.WaitReady("body"),
			chromedp.Sleep(config.WaitTime),
		)

		var companyURL string
		chromedp.Run(ctx,
			chromedp.AttributeValue(`div.group a`, "href", &companyURL, nil),
		)
		
		results = append(results, CompanyResult{
			Name: name,
			URL:  companyURL,
		})

		chromedp.Run(ctx,
			chromedp.Navigate(sj.URL),
			chromedp.Sleep(sj.WaitTime),
			chromedp.Nodes(sj.Selector, &nodes, chromedp.AtLeast(0)),
		)

	}

	return results, nil
}