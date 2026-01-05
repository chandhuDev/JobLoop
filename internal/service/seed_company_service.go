package service

import (
	"context"
	"log/slog"
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
		PWg:        &sync.WaitGroup{},
		YCWg:       &sync.WaitGroup{},
		ResultChan: make(chan models.SeedCompanyResult, 100),
		YCNodes:    make([]*cdp.Node, 0),
		PNodes:     make([]*cdp.Node, 0),
	}
}

func (s *SeedCompanyService) SeedCompanyConfigs(scraper *interfaces.ScraperClient) {
	slog.Info("seed company scraper started ")

	for i := 0; i < len(s.SeedCompany.Companies); i++ {

		if s.SeedCompany.Companies[i].Name == "Peer list" {
			s.SeedCompany.PWg.Add(1)

			go func(sp models.SeedCompany) {
				defer s.SeedCompany.PWg.Done()

				s.GetSeedCompaniesFromPeerList(scraper, &sp)
			}(s.SeedCompany.Companies[i])
		} else {
			s.SeedCompany.YCWg.Add(1)

			go func(yc models.SeedCompany) {
				defer s.SeedCompany.YCWg.Done()
				tabContext, tabCancel := scraper.Browser.RunInNewTab()
				s.GetSeedCompaniesFromYCombinator(tabContext, tabCancel, scraper, &yc)
			}(s.SeedCompany.Companies[i])
		}
	}
	go func() {
		s.SeedCompany.PWg.Wait()
		s.SeedCompany.YCWg.Wait()
		close(s.SeedCompany.ResultChan)
		slog.Info("closing seedcompany waitgroups and result channel")
	}()
}

func (s *SeedCompanyService) GetSeedCompaniesFromPeerList(scraper *interfaces.ScraperClient, sp *models.SeedCompany) {
	slog.Info("worker started for peerlist")

	tabContext, tabCancel := scraper.Browser.RunInNewTab()
	defer tabCancel()

	searchEngineKey := os.Getenv("GOOGLE_SEARCH_ENGINE")
	namesChan := make(chan string, 50)

	// --- Fetch nodes ---
	err := chromedp.Run(tabContext,
		chromedp.Navigate(sp.URL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(2*time.Second),
		chromedp.Nodes(sp.Selector, &s.SeedCompany.PNodes, chromedp.AtLeast(0)),
	)
	if err != nil {
		scraper.Err.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error fetching peerlist nodes",
			Err:      err,
		})
		return
	}

	slog.Info("Found nodes with selector",
		slog.Int("length", len(s.SeedCompany.PNodes)),
		slog.String("selector", sp.Selector),
	)

	// --- Search workers ---
	var searchWg sync.WaitGroup
	workerCount := 3

	for i := 0; i < workerCount; i++ {
		searchWg.Add(1)
		 slog.Info("Worker goroutine for peerlist", slog.Int("id", i))

		go func(workerID int) {
			defer searchWg.Done()

			for name := range namesChan {
				result, err := scraper.Search.SearchKeyWordInGoogle(
					name, workerID, searchEngineKey,
				)
				if err != nil {
					scraper.Err.Send(models.WorkerError{
						WorkerId: workerID,
						Message:  "error searching google",
						Err:      err,
					})
					continue
				}

				slog.Info("search results",
					slog.String("url", result),
				)

				s.SeedCompany.ResultChan <- models.SeedCompanyResult{
					CompanyName: name,
					CompanyURL:  result,
				}
			}
		}(i)
	}

	// --- Producer ---
	go func() {
		defer close(namesChan)

		for _, node := range s.SeedCompany.PNodes {
			var urlText string
			pXPath := node.FullXPath() + "/div[2]//p"

			err := chromedp.Run(tabContext,
				chromedp.WaitReady(pXPath, chromedp.BySearch),
				chromedp.Text(pXPath, &urlText, chromedp.BySearch),
			)
			if err != nil {
				continue
			}

			name := lastWord(urlText)
			// slog.Info("Uri in peerNodes", slog.String("uri", name))
			namesChan <- name
		}
	}()

	// --- Wait for workers ---
	searchWg.Wait()
}

func (s *SeedCompanyService) GetSeedCompaniesFromYCombinator(context context.Context, cancel context.CancelFunc, scraper *interfaces.ScraperClient, yc *models.SeedCompany) {
	slog.Info("worker started for ycombinator")
	defer cancel()

	chromedp.Run(context,
		chromedp.Navigate(yc.URL),
		chromedp.Sleep(yc.WaitTime),
		chromedp.Nodes(yc.Selector, &s.SeedCompany.YCNodes, chromedp.AtLeast(0)),
	)
	 slog.Info("Found nodes with selector", slog.Int("length of nodes", len(s.SeedCompany.YCNodes)), slog.String("for slector", yc.Selector), slog.String("for url", yc.URL))

	for i := range s.SeedCompany.YCNodes {

		var name string

		chromedp.Run(context,
			chromedp.Text(s.SeedCompany.YCNodes[i].FullXPath(), &name, chromedp.NodeVisible),
		)
		_, err := chromedp.RunResponse(context,
			chromedp.Click(s.SeedCompany.YCNodes[i].FullXPath()),
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
		 slog.Info("seed company Ycombinator", slog.String("CompanyName", name), slog.String("CompanyURL", url))

		chromedp.Run(context,
			chromedp.Navigate(yc.URL),
			chromedp.Sleep(yc.WaitTime),
			chromedp.Nodes(yc.Selector, &s.SeedCompany.YCNodes, chromedp.AtLeast(0)),
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
