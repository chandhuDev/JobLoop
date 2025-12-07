package browserscraper

import (
	"fmt"
	"context"
	"sync"
	"github.com/chromedp/chromedp"
)

func LaunchBrowser() {
	seedCompanies := []string{"producthunt", "ycombinator"}
	fmt.Println("Launching browser...")
	browserContext, browserCancel := chromedp.NewContext(context.Background())
	defer browserCancel()

	if err := chromdp.Run(browserContext); err!= nil {
		panic(err)
	}

	var wg sync.WaitGroup

    for i := range 2 {
		wg.Add(1)

		tabContext, tabCancel := chromedp.NewContext(browserContext)
		switch seedCompanies[i] {
		case "producthunt":
			go func() {
				defer wg.Done()
				defer tabCancel()
				openProductHuntSearchJobs(tabContext)
				
			}()
		case "ycombinator":
			go func() {
				defer wg.Done()
				defer tabCancel()
				openYCombinatorSearchJobs(tabContext)
				
			}()

		}
		
	}
    wg.Wait()
}

func openProductHuntSearchJobs (tabContext context.Context) {
   if err := chromedp.Run(tabContext, chromedp.Navigate("https://www.producthunt.com")); err!= nil {
	   fmt.Println("Error navigating to Product Hunt:", err)
   }
}

func openYCombinatorSearchJobs (tabContext context.Context) {
	if err:= chromedp.Run(tabContext, chromedp.Navigate("https://www.ycombinator.com/companies")); err!= nil {
		fmt.Println("Error navigating to Y Combinator:", err)
	}
}