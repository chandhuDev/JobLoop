package service

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/chandhuDev/JobLoop/internal/browser"
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

func NewSeedCompanyScraper(companyConfig SeedCompanyConfig) *SeedCompanyConfig {
	return &SeedCompanyConfig{
		Name:     companyConfig.Name,
		URL:      companyConfig.URL,
		Selector: companyConfig.Selector,
		WaitTime: companyConfig.WaitTime,
	}
}

func SeedCompanyConfigs(browser *browser.Browser, scc []SeedCompanyConfig) []SeedCompanyResult {
	var wg sync.WaitGroup
	var seedCompanyResults []SeedCompanyResult

	for i := 0; i < len(scc); i++ {
		wg.Add(1)
		go func(sc SeedCompanyConfig) {
			defer wg.Done()

			scraper := NewSeedCompanyScraper(sc)
			seedCompanyResults = scraper.ScrapeSeedCompanies(browser)

		}(scc[i])
	}
	wg.Wait()
	return seedCompanyResults
}

func (sc *SeedCompanyConfig) ScrapeSeedCompanies(b *browser.Browser) []SeedCompanyResult {
	var nodes []*cdp.Node
	var results []SeedCompanyResult
	var names []string
	var wg sync.WaitGroup

	fmt.Println("Scraping seed companies for:", sc)
	if sc.Name == "Peer list" {
		tabContext, tabCancel := b.RunInNewTab()
		defer tabCancel()
		chromedp.Run(tabContext,
			chromedp.Navigate(sc.URL),
			chromedp.Sleep(sc.WaitTime),
			chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
		)
		for i := 0; i < len(nodes); i++ {
			var name string
			pXPath := nodes[i].FullXPath() + "/div[2]//p"

			err := chromedp.Run(tabContext,
				chromedp.Text(pXPath, &name, chromedp.NodeVisible),
			)
			if err != nil {
				continue
			}
			if i == 3 {
				break
			}
			names = append(names, lastWord(name))

		}
		ch := make(chan SeedCompanyResult, len(names))

		for _, companyName := range names {
			wg.Add(1)
			go func(c string) {
				var companyURL string

				defer wg.Done()
				newtab, newcancel := b.RunInNewTab()
				defer newcancel()
				chromedp.Run(newtab,
					chromedp.Navigate("https://www.google.com"),
					chromedp.WaitVisible(`textarea[name="q"]`, chromedp.ByQuery),
					chromedp.SendKeys(`textarea[name="q"]`, c+"\n", chromedp.ByQuery),
					chromedp.WaitVisible(`div#search`, chromedp.ByQuery),

					chromedp.Sleep(5*time.Second),
					chromedp.Click(`(//div[@id='search']//a[h3])[1]`, chromedp.BySearch),
					chromedp.Sleep(3*time.Second),

					chromedp.Location(&companyURL),
				)
				ch <- SeedCompanyResult{
					CompanyURL:  companyURL,
					CompanyName: c,
				}
			}(companyName)

		}
		go func() {
			wg.Wait()
			close(ch)
		}()
		for result := range ch {
			fmt.Println("Peer list result:", result)
			results = append(results, result)
		}

	} else {
		tabContext, tabCancel := b.RunInNewTab()
		defer tabCancel()
		chromedp.Run(tabContext,
			chromedp.Navigate(sc.URL),
			chromedp.Sleep(sc.WaitTime),
			chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
		)
		fmt.Printf("Found %d nodes with selector '%s' on %s\n", len(nodes), sc.Selector, sc.URL)

		for i := 0; i < len(nodes); i++ {
			fmt.Println("results in loop", results)

			var name string

			if sc.Name == "Peer list" {

			} else {
				chromedp.Run(tabContext,
					chromedp.Text(nodes[i].FullXPath(), &name, chromedp.NodeVisible),
				)
				_, err := chromedp.RunResponse(tabContext,
					chromedp.Click(nodes[i].FullXPath()),
				)
				if err != nil {
					fmt.Println("error in clicking testimonial link:", err)
				}

				var url string
				err2 := chromedp.Run(tabContext,
					chromedp.AttributeValue(`div.group a`, "href", &url, nil),
				)
				if err2 != nil {
					fmt.Println("error in getting testimonial url:", err2)
				}
				results = append(results, SeedCompanyResult{
					CompanyName: name,
					CompanyURL:  url,
				})

				fmt.Println("Clicked on company:", name, "URL:", url)

				if i == 2 {
					break
				}

				chromedp.Run(tabContext,
					chromedp.Navigate(sc.URL),
					chromedp.Sleep(sc.WaitTime),
					chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
				)
			}

		}
		fmt.Println("results ", results)
	}

	return results
}

func lastWord(text string) string {
	fields := strings.Fields(text)
	if len(fields) == 0 {
		return ""
	}
	return fields[len(fields)-1]
}
