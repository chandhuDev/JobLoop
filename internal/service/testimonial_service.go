package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func ScrapeTestimonial(browser *browser.Browser, vision VisionConfig, seedCompanyList []SeedCompanyResult) {
	var nodes []*cdp.Node
	var wg sync.WaitGroup
	xpath := `
	(
	  //*[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'trust')]
		/following::*[count(.//img) >= 3][1]
	)//img
	`
	for i := 0; i < len(seedCompanyList); i++ {
		tabContext, tabCancel := browser.RunInNewTab()

		err := chromedp.Run(tabContext,
			chromedp.Navigate(seedCompanyList[i].CompanyURL),
			chromedp.WaitVisible("body"),
			chromedp.Nodes(xpath, &nodes, chromedp.BySearch, chromedp.AtLeast(0)),
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
			fullURL = getAttr(tabContext, nodes[j].FullXPath(), "src")
			fmt.Println("Found testimonial image URL:", fullURL)
			if fullURL == "" || fullURL == "null" {
				fullURL = getAttr(tabContext, nodes[j].FullXPath(), "data-src")
			}
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

func getAttr(ctx context.Context, xpath string, attributeName string) string {
	var url string
	// chromedp.Run(ctx, chromedp.AttributeValue(xpath, attributeName, &url, nil, chromedp.BySearch))
	// if url != "" {
	// 	fmt.Println("Error getting attribute value from nodes", url)
	// 	return url
	// }
	chromedp.Run(ctx, chromedp.JavascriptAttribute(xpath, attributeName, &url))
     fmt.Println("Extracted attribute", attributeName, "with value:", url)
	return url
}
