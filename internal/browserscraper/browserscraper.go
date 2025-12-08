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

	opts := append(chromedp.DefaultExecAllocatorOptions[:],
		chromedp.DisableGPU,
		chromedp.UserDataDir(dir),
	)
	allocContext, allocCancel := chromedp.NewExecAllocator(context.Background(), opts...)
	defer allocCancel()

	if err := chromdp.Run(allocContext); err!= nil {
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
	var productLinks []*cdp.Node
    var results []string

    err := chromedp.Run(tabContext,
	 chromedp.Navigate("https://www.producthunt.com")
	 chromedp.Sleep(2*time.Second)
	 chromedp.Nodes(`a[href^="/products/"]`, &productLinks, chromedp.AtLeast(0)),
	 ); 
	 
	 for _, n := range nodes {
		var text string
		err := chromedp.Run(ctx,
			chromedp.Text(n.FullXPath(), &text, chromedp.NodeVisible),
		)
		if err != nil {
			continue
		}

		results = append(results, text)
	}

	for i, r := range results {
		fmt.Printf("%d → %s\n", i+1, r)
	}
}


func openYCombinatorSearchJobs (tabContext context.Context) {
	var productLinks []*cdp.Node
    var resutls []string

	err:= chromedp.Run(tabContext, 
		chromedp.Navigate("https://www.ycombinator.com/companies")
		chromedp.Sleep(5*time.Second)
		chromedp.Nodes(`a[href^="/comapnies/"]`, &productLinks, chromedp.AtLeast(0)),

	);
     
	if err!= nil {
		fmt.Println("Error navigating to Y Combinator:", err)
	}

	for _, n := range nodes {
		var text string
		err := chromedp.Run(ctx,
			chromedp.Text(n.FullXPath(), &text, chromedp.NodeVisible),
		)
		if err != nil {
			continue
		}

		results = append(results, text)
	}

	for i, r := range results {
		fmt.Printf("%d → %s\n", i+1, r)
	}

}