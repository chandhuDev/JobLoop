package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chandhuDev/JobLoop/internal/Utils/error"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

func ScrapeTestimonial(browser *browser.Browser, vision VisionConfig, scChan <-chan SeedCompanyResult, e chan error.WorkerError) {
	var testimonialWG sync.WaitGroup
	var imageWG sync.WaitGroup
	imageResultChan := make(chan []string, 100)

	for i := 0; i < 5; i++ {
		testimonialWG.Add(1)
		go func(i int, browser browser.Browser, scChan <-chan SeedCompanyResult, wg *sync.WaitGroup, im chan []string) {
			fmt.Printf("Starting testimonial scraper goroutine %d\n", i)

			defer testimonialWG.Done()
			tabContext, tabCancel := browser.RunInNewTab()
			defer tabCancel()
			xpath := `
		  (
			//*[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'trust')]
			  /following::*[count(.//img) >= 3][1]
		  )//img
		  `

			for scr := range scChan {
				var nodes []*cdp.Node

				err := chromedp.Run(tabContext,
					chromedp.Navigate(scr.CompanyURL),
					chromedp.WaitVisible("body"),
					chromedp.Nodes(xpath, &nodes, chromedp.BySearch, chromedp.AtLeast(0)),
				)
				if err != nil {
					fmt.Println("Error navigating to testimonial page:", err)
				}
				if len(nodes) == 0 || nodes == nil {
					fmt.Println("No testimonial images found for", scr.CompanyName)
					break
				}
				var UrlArray []string

				for j := range nodes {
					var fullURL string
					fullURL = getAttr(tabContext, nodes[j].FullXPath(), "src")
					if fullURL == "" || fullURL == "null" {
						fullURL = getAttr(tabContext, nodes[j].FullXPath(), "data-src")
					}
					UrlArray = append(UrlArray, fullURL)
				}
				im <- UrlArray

			}
		}(i, browser, scChan, &testimonialWG, imageResultChan)

	}

	for i := 0; i < 5; i++ {
		imageWG.Add(1)
		go func(i int, v VisionConfig) {
			fmt.Printf("Starting image processor goroutine %d\n", i)
			defer imageWG.Done()
			for urlArray := range imageResultChan {
				v.ExtractImageFromText(urlArray)

			}
		}(i, vision)
	}

	go func() {
		testimonialWG.Wait()
		close(imageResultChan)
	}()
	imageWG.Wait()

}

func getAttr(ctx context.Context, xpath string, attributeName string) string {
	var url string
	err := chromedp.Run(ctx, chromedp.JavascriptAttribute(xpath, attributeName, &url))
	if err != nil {
		fmt.Printf("Error getting JS attribute %s at %s: %v\n", attributeName, xpath, err)
		return ""
	}
	if url != "" {
		fmt.Printf("Extracted %s: %s\n", attributeName, url)
	}
	fmt.Println("Extracted attribute", attributeName, "with value:", url)
	return url
}
