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

func LaunchBrowser() {
	configs := []ScrapeConfig{
		{
			Name:     "Product Hunt",
			URL:      "https://www.producthunt.com",
			Selector: `a[href^="/products/"]`,
			WaitTime: 2 * time.Second,
		},
		{
			Name:     "Y Combinator",
			URL:      "https://www.ycombinator.com/companies",
			Selector: `a[href^="/companies/"]`,
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
			scrapeLinks(tabContext, cfg)
		}(config)
	}

	wg.Wait()
}

func scrapeLinks(ctx context.Context, config ScrapeConfig) {
	fmt.Printf("Opening %s...\n", config.Name)

	var nodes []*cdp.Node

	err := chromedp.Run(ctx,
		chromedp.Navigate(config.URL),
		chromedp.Sleep(config.WaitTime),
		chromedp.Nodes(config.Selector, &nodes, chromedp.AtLeast(0)),
	)

	if err != nil {
		fmt.Printf("Error navigating to %s: %v\n", config.Name, err)
		return
	}

	results := extractTextFromNodes(ctx, nodes)

	fmt.Printf("\n=== %s Results (%d found) ===\n", config.Name, len(results))
	for i, text := range results {
		fmt.Printf("%d → %s\n", i+1, text)
	}
}

func extractTextFromNodes(parentCtx context.Context, nodes []*cdp.Node) []string {
	var results []string

	for _, node := range nodes {
		var text string
		err := chromedp.Run(parentCtx,
			chromedp.Text(node.FullXPath(), &text, chromedp.NodeVisible),
		)
		if err != nil {
			continue
		}
		
		if text != "" {
			results = append(results, text)
		}
	}

	return results
}