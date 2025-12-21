package service

import (
	"fmt"
	"sync"
	"time"
	"context"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chandhuDev/JobLoop/internal/browser"
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
	return &SeedCompanyConfig{
		Name : companyConfig.Name, 
		URL : companyConfig.URL, 
		Selector : companyConfig.Selector, 
		WaitTime : companyConfig.WaitTime,
	}
}

func SeedCompanyConfigs(browser browser.Browser,scc []SeedCompanyConfig) []SeedCompanyResult {
	var wg sync.WaitGroup
	var seedCompanyResults []SeedCompanyResult

    for i:=0; i < len(scc); i++ {
		wg.Add(1)
		go func(sc SeedCompanyConfig) {
			defer wg.Done()
            tabContext, tabCancel := browser.RunInNewTab()
			defer tabCancel()
			scraper := NewSeedCompanyScraper(sc)
		    seedCompanyResults = scraper.ScrapeSeedCompanies(tabContext)
		    
		}(scc[i])
	}
	wg.Wait()
    return seedCompanyResults
}

func (sc *SeedCompanyConfig) ScrapeSeedCompanies(ctx context.Context) []SeedCompanyResult {
	var nodes []*cdp.Node
	var results []SeedCompanyResult

	fmt.Println("Scraping seed companies for:", sc.Name)

	chromedp.Run(ctx,
		chromedp.Navigate(sc.URL),
		chromedp.Sleep(sc.WaitTime),
		chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
	)

    for i:=0; i < len(nodes); i++ {
		
		var name string
		chromedp.Run(ctx,
			chromedp.Text(nodes[i].FullXPath(), &name, chromedp.NodeVisible),
		)

		_, err := chromedp.RunResponse(ctx,
			chromedp.Click(nodes[i].FullXPath()),
		)
		if err != nil {
			fmt.Println("error in clicking testimonial link:", err)
		}

		var url string
		err2 := chromedp.Run(ctx,
			chromedp.AttributeValue(`div.group a`, "href", &url, nil),
		)
		if err2!=nil {
			fmt.Println("error in getting testimonial url:", err2)
		}
        results = append(results, SeedCompanyResult{
			CompanyName: name,
			CompanyURL: url,
		})

		if i == 2 {
			break
		}

		chromedp.Run(ctx, 
			chromedp.NavigateBack(),
			chromedp.WaitVisible(sc.Selector),		
			chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
		)
      
	}
	return results
}