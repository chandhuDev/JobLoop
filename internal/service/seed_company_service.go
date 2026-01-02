package service

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type SeedCompanyService struct {
	SeedCompany *models.SeedCompanyArray
}

func NewSeedCompanyScraper(companyConfig models.SeedCompany) *models.SeedCompany {
	return &models.SeedCompany{
		Name:     companyConfig.Name,
		URL:      companyConfig.URL,
		Selector: companyConfig.Selector,
		WaitTime: companyConfig.WaitTime,
	}
}

func NewSeedCompanyArray(firstSeedCompany models.SeedCompany, secondSeedCompany models.SeedCompany) *models.SeedCompanyArray {
	return &models.SeedCompanyArray{
		Companies:  []models.SeedCompany{firstSeedCompany, secondSeedCompany},
		Wg:         &sync.WaitGroup{},
		ResultChan: make(chan models.SeedCompanyResult, 100),
		Nodes:      make([]*cdp.Node, 0),
	}
}

func (s *SeedCompanyService) SeedCompanyConfigs(scraper *interfaces.ScraperClient) {
	for i := 0; i < len(s.SeedCompany.Companies); i++ {
		if s.SeedCompany.Companies[i].Name == "Peer list" {
			s.SeedCompany.Wg.Add(1)

			go func(sp models.SeedCompany) {
				s.GetSeedCompaniesFromPeerList(scraper, &sp)
			}(s.SeedCompany.Companies[i])
		} else {
			s.SeedCompany.Wg.Add(1)

			go func(yc models.SeedCompany) {
				tabContext, tabCancel := scraper.Browser.RunInNewTab()
				s.GetSeedCompaniesFromYCombinator(tabContext, tabCancel, scraper, &yc)
			}(s.SeedCompany.Companies[i])
		}
	}
	go func() {
		defer s.SeedCompany.Wg.Done()
		close(s.SeedCompany.ResultChan)
	}()
}

func (s *SeedCompanyService) GetSeedCompaniesFromPeerList(scraper *interfaces.ScraperClient, sp *models.SeedCompany) {
	tabContext, tabCancel := scraper.Browser.RunInNewTab()
	defer tabCancel()
	serachEnginKey := os.Getenv("GOOGLE_SEARCH_ENGINE")
	namesChan := make(chan string)

	chromedp.Run(tabContext,
		chromedp.Navigate(sp.URL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second),
		chromedp.Nodes(sp.Selector, &s.SeedCompany.Nodes, chromedp.AtLeast(0)),
	)
	for i := 0; i < len(s.SeedCompany.Nodes); i++ {
		var Url string
		pXPath := s.SeedCompany.Nodes[i].FullXPath() + "/div[2]//p"

		err := chromedp.Run(tabContext,
			chromedp.WaitReady(pXPath, chromedp.BySearch),
			chromedp.Text(pXPath, &Url, chromedp.BySearch),
		)
		if err != nil {
			scraper.Err.Send(models.WorkerError{
				WorkerId: i,
				Message:  "error in collecting nodes:" + Url,
				Err:      err,
			})
			continue
		}
		namesChan <- lastWord(Url)
	}

	for i := 0; i < 3; i++ {
		s.SeedCompany.Wg.Add(1)

		go func(i int) {
			defer s.SeedCompany.Wg.Done()
			fmt.Printf("Worker %d processing\n", i)
			for name := range namesChan {
				result, err := scraper.Search.SearchKeyWordInGoogle(name, i, serachEnginKey)
				scraper.Err.Send(models.WorkerError{
					WorkerId: i,
					Message:  "error in collecting nodes:" + result,
					Err:      err,
				})
				fmt.Println("search results for ", name, ":", result)
				s.SeedCompany.ResultChan <- models.SeedCompanyResult{
					CompanyName: name,
					CompanyURL:  result,
				}
			}
		}(i)
	}
}

func (s *SeedCompanyService) GetSeedCompaniesFromYCombinator(context context.Context, cancel context.CancelFunc, scraper *interfaces.ScraperClient, yc *models.SeedCompany) {
	defer cancel()

	chromedp.Run(context,
		chromedp.Navigate(yc.URL),
		chromedp.Sleep(yc.WaitTime),
		chromedp.Nodes(yc.Selector, &s.SeedCompany.Nodes, chromedp.AtLeast(0)),
	)
	fmt.Printf("Found %d nodes with selector '%s' on %s\n", len(s.SeedCompany.Nodes), yc.Selector, yc.URL)

	for i := range s.SeedCompany.Nodes {

		var name string

		chromedp.Run(context,
			chromedp.Text(s.SeedCompany.Nodes[i].FullXPath(), &name, chromedp.NodeVisible),
		)
		_, err := chromedp.RunResponse(context,
			chromedp.Click(s.SeedCompany.Nodes[i].FullXPath()),
		)
		if err != nil {
			scraper.Err.Send(models.WorkerError{
				WorkerId: -1,
				Message:  "error in clicking testimonial link",
				Err:      err,
			})
		}

		var url string
		err2 := chromedp.Run(context,
			chromedp.AttributeValue(`div.group a`, "href", &url, nil),
		)
		if err2 != nil {
			scraper.Err.Send(models.WorkerError{
				WorkerId: -1,
				Message:  "error in getting testimonial url:",
				Err:      err2,
			})
		}
		s.SeedCompany.ResultChan <- models.SeedCompanyResult{
			CompanyName: name,
			CompanyURL:  url,
		}

		fmt.Println("Clicked on company:", name, "URL:", url)

		chromedp.Run(context,
			chromedp.Navigate(yc.URL),
			chromedp.Sleep(yc.WaitTime),
			chromedp.Nodes(yc.Selector, &s.SeedCompany.Nodes, chromedp.AtLeast(0)),
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
