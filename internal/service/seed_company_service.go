package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chandhuDev/JobLoop/internal/Utils/error"
	"github.com/chandhuDev/JobLoop/internal/config/search"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type SeedCompanyConfig struct {
	Name     string
	URL      string
	Selector string
	WaitTime time.Duration
}

type SeedCompanyResult struct {
	CompanyName string
	CompanyURL  string
}

func NewSeedCompanyscaraper(companyConfig SeedCompanyConfig) *SeedCompanyConfig {
	return &SeedCompanyConfig{
		Name:     companyConfig.Name,
		URL:      companyConfig.URL,
		Selector: companyConfig.Selector,
		WaitTime: companyConfig.WaitTime,
	}
}

func SeedCompanyConfigs(b *browser.Browser, scc []SeedCompanyConfig, scChan chan SeedCompanyResult, errChan chan error.WorkerError) {
	var wg sync.WaitGroup
	for i := 0; i < len(scc); i++ {
		if scc[i].Name == "Peer list" {
			wg.Add(1)

			go func(d SeedCompanyConfig, b *browser.Browser, scChan chan SeedCompanyResult, wg *sync.WaitGroup,	errChan chan error.WorkerError) {
				d.getSeedCompaniesFromPeerList(b, scChan, wg, errChan)
			}(scc[i], b, scChan, &wg, errChan)
		} else {
			wg.Add(1)

			go func(d SeedCompanyConfig, b *browser.Browser, scChan chan SeedCompanyResult, wg *sync.WaitGroup, errChan chan error.WorkerError) {
				tabContext, tabCancel := b.RunInNewTab()
				d.getSeedCompaniesFromYCombinator(tabContext, tabCancel, scChan, wg, errChan)
			}(scc[i], b, scChan, &wg, errChan)
		}
	}
	go func() {
		defer wg.Done()
		close(scChan)
	}()
}

func (sca *SeedCompanyConfig) getSeedCompaniesFromPeerList(b *browser.Browser, resultChannel chan SeedCompanyResult, wg *sync.WaitGroup, e chan error.WorkerError) {
	tabContext, tabCancel := b.RunInNewTab()
	defer tabCancel()
	var nodes []*cdp.Node
	serachEnginKey := os.Getenv("GOOGLE_SEARCH_ENGINE")
	customSearchInstance := search.CreateSearchService(context.Background())
	namesChan := make(chan string)

	chromedp.Run(tabContext,
		chromedp.Navigate(sca.URL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second),
		chromedp.Nodes(sca.Selector, &nodes, chromedp.AtLeast(0)),
	)
	for i := 0; i < len(nodes); i++ {
		var Url string
		pXPath := nodes[i].FullXPath() + "/div[2]//p"

		err := chromedp.Run(tabContext,
			chromedp.WaitReady(pXPath, chromedp.BySearch),
			chromedp.Text(pXPath, &Url, chromedp.BySearch),
		)
		if err != nil {
			e <- error.WorkerError{
				WorkerId: i,
				Message: "error in collecting nodes:" + Url,
				Err : err,
			}
			continue
		}
		namesChan <- lastWord(Url)
	}

	for i := 0; i < 3; i++ {
		wg.Add(1)

		go func(i int) {
			defer wg.Done()
			fmt.Printf("Worker %d processing\n", i)
			for name := range namesChan {
				v, err := customSearchInstance.Cse.List().Q(name).Cx(serachEnginKey).Do()
				if e != nil {
					e <- error.WorkerError{
						WorkerId: i,
						Message: "error in searching company name:" + name,
						Err : err,
					}
				}
				fmt.Println("search results for ", name, ":", v.Items[0].Link)
				resultChannel <- SeedCompanyResult{
					CompanyName: name,
					CompanyURL:  v.Items[0].Link,
				}

			}

		}(i)
	}
}

func (sca *SeedCompanyConfig) getSeedCompaniesFromYCombinator(tabContext context.Context, tabCancel context.CancelFunc, resultChan chan SeedCompanyResult, wg *sync.WaitGroup, e chan error.WorkerError) {
	defer tabCancel()
	var nodes []*cdp.Node

	chromedp.Run(tabContext,
		chromedp.Navigate(sca.URL),
		chromedp.Sleep(sca.WaitTime),
		chromedp.Nodes(sca.Selector, &nodes, chromedp.AtLeast(0)),
	)
	fmt.Printf("Found %d nodes with selector '%s' on %s\n", len(nodes), sca.Selector, sca.URL)

	for i := range nodes {

		var name string

		chromedp.Run(tabContext,
			chromedp.Text(nodes[i].FullXPath(), &name, chromedp.NodeVisible),
		)
		_, err := chromedp.RunResponse(tabContext,
			chromedp.Click(nodes[i].FullXPath()),
		)
		if err != nil {
			e <- error.WorkerError{
				WorkerId: -1,
				Message: "error in clicking testimonial link",
				Err : err,
			}
		}

		var url string
		err2 := chromedp.Run(tabContext,
			chromedp.AttributeValue(`div.group a`, "href", &url, nil),
		)
		if err2 != nil {
			e <- error.WorkerError{
				WorkerId: -1,
				Message: "error in getting testimonial url:",
				Err : err2,
			}
		}
		resultChan <- SeedCompanyResult{
			CompanyName: name,
			CompanyURL:  url,
		}

		fmt.Println("Clicked on company:", name, "URL:", url)

		chromedp.Run(tabContext,
			chromedp.Navigate(sca.URL),
			chromedp.Sleep(sca.WaitTime),
			chromedp.Nodes(sca.Selector, &nodes, chromedp.AtLeast(0)),
		)
	}

}

func lastWord(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}

	for i, field := range fields {
		if strings.ToLower(field) == "at" && i+1 < len(fields) {
			return strings.Join(fields[i+1:], " ")
		}
	}

	return fields[len(fields)-1]
}
