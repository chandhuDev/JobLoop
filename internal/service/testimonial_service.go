package service

import (
	"fmt"
	"sync"

	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func ScrapeTestimonial(browser *browser.Browser, vision VisionConfig, seedCompanyList []SeedCompanyResult) {
	var nodes []*cdp.Node
	var wg sync.WaitGroup

	for i := 0; i < len(seedCompanyList); i++ {
		tabContext, tabCancel := browser.RunInNewTab()

		err := chromedp.Run(tabContext,
			chromedp.Navigate(seedCompanyList[i].CompanyURL),
			chromedp.WaitVisible("body"),
			chromedp.Nodes(`//*[contains(text(), "Trusted by")]/ancestor::*[count(.//img) > 1][1]//img`, &nodes, chromedp.AtLeast(0)),
		)

		if err != nil {
			fmt.Println("Error navigating to testimonial page:", err)
			tabCancel()
		}

		if len(nodes) == 0 || nodes == nil {
			fmt.Println("No testimonial images found for", seedCompanyList[i].CompanyName)
			tabCancel()
			continue
		}

		var requestsArray []string
		for j := 0; j < len(nodes); j++ {
			var fullURL string
			chromedp.Run(tabContext,
				chromedp.JavascriptAttribute(nodes[j].FullXPath(), "src", &fullURL),
			)
			requestsArray = append(requestsArray, fullURL)
		}

		tabCancel()

		wg.Add(1)
		go func(urls []string) {
			defer wg.Done()
			vision.ExtractImageFromText(urls)
		}(requestsArray)
	}
	wg.Wait()
}
