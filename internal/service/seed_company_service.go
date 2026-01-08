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

func (s *SeedCompanyService) SeedCompanyConfigs(ctx context.Context, scraper *interfaces.ScraperClient) {
	slog.Info("seed company scraper started ")
	defer close(s.SeedCompany.ResultChan)

	for i := 0; i < len(s.SeedCompany.Companies); i++ {

		if s.SeedCompany.Companies[i].Name == "Peer list" {
			s.SeedCompany.PWg.Add(1)

			go func(sp models.SeedCompany) {
				defer s.SeedCompany.PWg.Done()

				s.GetSeedCompaniesFromPeerList(scraper, &sp, ctx)
			}(s.SeedCompany.Companies[i])
		} else {
			s.SeedCompany.YCWg.Add(1)

			go func(yc models.SeedCompany) {
				defer s.SeedCompany.YCWg.Done()
				s.GetSeedCompaniesFromYCombinator(ctx, scraper, &yc)
			}(s.SeedCompany.Companies[i])
		}
	}

	s.SeedCompany.PWg.Wait()
	s.SeedCompany.YCWg.Wait()
	slog.Info("closing seedcompany waitgroups and result channel")
}

func (s *SeedCompanyService) GetSeedCompaniesFromPeerList(scraper *interfaces.ScraperClient, sp *models.SeedCompany, ctx context.Context) {

	// select {
	// case <-ctx.Done():
	// 	return
	// default:
	// }
	slog.Info("worker started for peerlist")

	tabContext, tabCancel := scraper.Browser.RunInNewTab()
	defer tabCancel()

	searchEngineKey := os.Getenv("GOOGLE_SEARCH_ENGINE")
	namesChan := make(chan string, 50)
	slog.Info("START processing for peerlist", slog.Time("time", time.Now()))

	err := chromedp.Run(tabContext,
		chromedp.Navigate(sp.URL),
		chromedp.WaitReady("body"),
		chromedp.Sleep(4*time.Second),
		chromedp.Nodes(sp.Selector, &s.SeedCompany.PNodes, chromedp.AtLeast(0)),
	)
	if err != nil {
		scraper.Err.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error fetching peerlist nodes",
			Err:      err,
		})
	}
	slog.Info("END processing for peerlist", slog.Time("time", time.Now()))

	slog.Info("Found nodes with selector in peerlist",
		slog.Int("length", len(s.SeedCompany.PNodes)),
		slog.String("selector", sp.Selector),
	)

	go func() {
		defer close(namesChan)

		for i := 0; i < len(s.SeedCompany.PNodes); i++ {
			slog.Info("sending names to namesChan")

			if i == 3 {
				break
			}
			var urlText string

			pXPath := s.SeedCompany.PNodes[i].FullXPath() + "/div[2]//p"

			err := chromedp.Run(tabContext,
				chromedp.WaitReady(pXPath, chromedp.BySearch),
				chromedp.Text(pXPath, &urlText, chromedp.BySearch),
			)
			if err != nil {
				continue
			}
			namesChan <- lastWord(urlText)

		}
	}()

	var searchWg sync.WaitGroup
	workerCount := 3

	for i := 0; i < workerCount; i++ {
		searchWg.Add(1)

		go func(workerID int) {
			defer searchWg.Done()

			for name := range namesChan {
				slog.Info("starting goroutine for peerlist search scraper", slog.Int("id", i))

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
				s.SeedCompany.ResultChan <- models.SeedCompanyResult{
					CompanyName: name,
					CompanyURL:  result,
				}
			}
		}(i)
	}

	searchWg.Wait()
}

func (s *SeedCompanyService) GetSeedCompaniesFromYCombinator(ctx context.Context, scraper *interfaces.ScraperClient, yc *models.SeedCompany) {
    slog.Info("worker started for ycombinator")
    
    tabContext, tabCancel := scraper.Browser.RunInNewTab()
    defer tabCancel()

    chromedp.Run(tabContext,
        chromedp.Navigate(yc.URL),
        chromedp.Sleep(yc.WaitTime),
        chromedp.Nodes(yc.Selector, &s.SeedCompany.YCNodes, chromedp.AtLeast(0)),
    )
    
    slog.Info("Found nodes", slog.Int("length", len(s.SeedCompany.YCNodes)))

    for i := 0; i < 2; i++ {
        if i >= len(s.SeedCompany.YCNodes) {
            slog.Warn("Not enough nodes", slog.Int("have", len(s.SeedCompany.YCNodes)), slog.Int("need", i))
            break
        }

        var url string
        var name string

        chromedp.Run(tabContext,
            chromedp.Text(s.SeedCompany.YCNodes[i].FullXPath(), &name, chromedp.NodeVisible),
        )
        
        chromedp.RunResponse(tabContext,
            chromedp.Click(s.SeedCompany.YCNodes[i].FullXPath()),
        )

        chromedp.Run(tabContext,
            chromedp.AttributeValue(`div.group a`, "href", &url, nil),
        )
        
        slog.Info("seed company Ycombinator", slog.String("CompanyName", name), slog.String("CompanyURL", url))

        s.SeedCompany.ResultChan <- models.SeedCompanyResult{
            CompanyName: name,
            CompanyURL:  url,
        }

        chromedp.Run(tabContext,
            chromedp.Navigate(yc.URL),
            chromedp.Sleep(yc.WaitTime),
            chromedp.Nodes(yc.Selector, &s.SeedCompany.YCNodes, chromedp.AtLeast(0)),
        )
        
        if len(s.SeedCompany.YCNodes) == 0 {
            slog.Warn("No nodes after re-navigation")
            break
        }
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
