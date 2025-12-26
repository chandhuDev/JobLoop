package service

import (
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
	"google.golang.org/api/customsearch/v1"
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

func SeedCompanyConfigs(browser *browser.Browser, scc []SeedCompanyConfig, search *customsearch.Service) []SeedCompanyResult {
	var wg sync.WaitGroup
	var seedCompanyResults []SeedCompanyResult

	for i := 0; i < len(scc); i++ {
		wg.Add(1)
		go func(sc SeedCompanyConfig) {
			defer wg.Done()

			scraper := NewSeedCompanyScraper(sc)
			seedCompanyResults = scraper.ScrapeSeedCompanies(browser, search)
			fmt.Println("Scraping results of seed companies:", seedCompanyResults)

		}(scc[i])
	}
	wg.Wait()
	return seedCompanyResults
}

func (sc *SeedCompanyConfig) ScrapeSeedCompanies(b *browser.Browser, search *customsearch.Service) []SeedCompanyResult {
	var nodes []*cdp.Node
	var results []SeedCompanyResult
	var names []string
	engine := os.Getenv("GOOGLE_SEARCH_ENGINE")

	fmt.Println("Scraping seed companies for:", sc)
	if sc.Name == "Peer list" {
		tabContext, tabCancel := b.RunInNewTab()
		defer tabCancel()
		chromedp.Run(tabContext,
			chromedp.Navigate(sc.URL),
			chromedp.WaitReady("body"),
			chromedp.Sleep(2*time.Second),
			chromedp.Nodes(sc.Selector, &nodes, chromedp.AtLeast(0)),
		)
		for i := 0; i < len(nodes); i++ {
			var name string
			pXPath := nodes[i].FullXPath() + "/div[2]//p"

			err := chromedp.Run(tabContext,
				chromedp.WaitReady(pXPath, chromedp.BySearch),
				chromedp.Text(pXPath, &name, chromedp.BySearch),
			)
			if err != nil {
				continue
			}
			if i == 3 {
				break
			}
			names = append(names, lastWord(name))

		}

		fmt.Println("searching for company:", len(names))

		for _, c := range names {
			v, e := search.Cse.List().Q(c).Cx(engine).Do()
			if e != nil {
				fmt.Println("error in search:", e)
			}
			fmt.Println("search results for ", c, ":", v.Items[0].Link)
			results = append(results, SeedCompanyResult{
				CompanyName: c,
				CompanyURL:  v.Items[0].Link,
			})
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

		fmt.Println("results ", results)
	}

	return results
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
