package browserscraper

import (
	"fmt"
	"context"
	"sync"
	"time"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
)

type ScrapeConfig struct {
	Name     string
	URL      string
	Selector string
	WaitTime time.Duration
}

type CompanyResult struct {
    Name string
	URL string
}


func LaunchBrowser() {
	configs := []ScrapeConfig{
	    {
			Name:     "Y Combinator",
			URL:      "https://www.ycombinator.com/companies",
			Selector: `span[class^="_coName_i9oky_470"]`,
			WaitTime: 5 * time.Second,
		},
	}

	fmt.Println("Launching browser...")

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.Flag("headless", false),
		chromedp.Flag("disable-gpu", false),
		chromedp.WindowSize(1920, 1080),
	)

	allocContext, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	browserContext, browserCancel := chromedp.NewContext(allocContext)
	defer browserCancel()

	if err := chromedp.Run(browserContext); err != nil {
		panic(err)
	}

	var wg sync.WaitGroup

	for _, config := range configs {
		wg.Add(1)
		tabContext, tabCancel := chromedp.NewContext(browserContext)

		go func(cfg ScrapeConfig) {
			defer wg.Done()
			defer tabCancel()
			results:=scrapeLinks(tabContext, cfg)
		}(config)
	}

	wg.Wait()
}

func scrapeLinks(ctx context.Context, config ScrapeConfig) []CompanyResult {
	var nodes []*cdp.Node
    
	err := chromedp.Run(ctx,
		chromedp.Navigate(config.URL),
		chromedp.Sleep(config.WaitTime),
		chromedp.Nodes(config.Selector, &nodes, chromedp.AtLeast(0)),
	)

	if err != nil {
		fmt.Printf("Error navigating to %s: %v\n", config.Name, err)
		return []CompanyResult{}
	}

	var results []CompanyResult

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

		if i == 1 {
			break
		}

		chromedp.Run(ctx,
			chromedp.Navigate("https://www.ycombinator.com/companies"),
			chromedp.Sleep(config.WaitTime),
			chromedp.Nodes(config.Selector, &nodes, chromedp.AtLeast(0)),
		)

	}
	return results
}