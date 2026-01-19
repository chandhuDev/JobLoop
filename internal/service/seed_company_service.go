package service

import (
	"context"
	"log/slog"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
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
	}
}

func (s *SeedCompanyService) SeedCompanyConfigs(ctx context.Context, scraper *interfaces.ScraperClient) {
	slog.Info("seed company scraper started")
	defer close(s.SeedCompany.ResultChan)

	for i := 0; i < len(s.SeedCompany.Companies); i++ {
		if s.SeedCompany.Companies[i].Name == "Peer list" {
			s.SeedCompany.PWg.Add(1)
			go func(sp models.SeedCompany) {
				defer s.SeedCompany.PWg.Done()
				s.GetSeedCompaniesFromPeerList(scraper, &sp, ctx)
			}(s.SeedCompany.Companies[i])
		} else {
			slog.Info("in ycomnibator go routine")
			slog.Info("in ycomnibator go routine")
			slog.Info("in ycomnibator go routine")
			slog.Info("in ycomnibator go routine")

			// s.SeedCompany.YCWg.Add(1)
			// go func(yc models.SeedCompany) {
			// 	defer s.SeedCompany.YCWg.Done()
			// 	s.GetSeedCompaniesFromYCombinator(ctx, scraper, &yc)
			// }(s.SeedCompany.Companies[i])
		}
	}

	s.SeedCompany.PWg.Wait()
	s.SeedCompany.YCWg.Wait()
	slog.Info("closing seedcompany waitgroups and result channel")
}

func (s *SeedCompanyService) GetSeedCompaniesFromPeerList(scraper *interfaces.ScraperClient, sp *models.SeedCompany, ctx context.Context) {
	slog.Info("worker started for peerlist")

	page, err := scraper.Browser.RunInNewTab()
	if err != nil {
		scraper.Err.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error creating page for peerlist",
			Err:      err,
		})
		return
	}
	defer page.Close()

	searchEngineKey := os.Getenv("GOOGLE_SEARCH_ENGINE")
	namesChan := make(chan string, 50)
	slog.Info("START processing for peerlist", slog.Time("time", time.Now()))

	if _, err := page.Goto(sp.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		scraper.Err.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error navigating to peerlist",
			Err:      err,
		})
		return
	}

	locator := page.Locator(sp.Selector)
	count, err := locator.Count()
	if err != nil {
		scraper.Err.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "error getting peerlist nodes count",
			Err:      err,
		})
		return
	}

	slog.Info("Found nodes with selector in peerlist",
		slog.Int("length", count),
		slog.String("selector", sp.Selector),
	)

	go func() {
		defer close(namesChan)

		for i := 0; i < count && i < 3; i++ {
			slog.Info("sending names to namesChan")

			item := locator.Nth(i)
			pElement := item.Locator("div:first-child > p")

			urlText, err := pElement.TextContent()
			if err != nil {
				slog.Error("error getting text", slog.Any("error", err))
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
				slog.Info("starting goroutine for peerlist search scraper", slog.Int("id", workerID))

				result, err := scraper.Search.SearchKeyWordInGoogle(
					name, workerID, searchEngineKey,
				)
				slog.Info("company result", slog.String("url", result))

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

	page, err := scraper.Browser.RunInNewTab()
	if err != nil {
		slog.Error("error creating page for ycombinator", slog.Any("error", err))
		return
	}
	defer page.Close()

	if _, err := page.Goto(yc.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		slog.Error("error navigating to ycombinator", slog.Any("error", err))
		return
	}

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	locator := page.Locator(yc.Selector)
	count, _ := locator.Count()
	slog.Info("Found companies", slog.Int("count", count))

	for i := 0; i < 2; i++ {
		if i >= count {
			slog.Warn("Not enough nodes in Ycombinator", slog.Int("have", count), slog.Int("need", i))
			break
		}

		item := locator.Nth(i)

		name, err := item.TextContent()
		if err != nil {
			slog.Error("error getting at YC", slog.Any("error", err))
			continue
		}

		if err := item.Click(); err != nil {
			slog.Error("error clicking item at YC", slog.Any("error", err))
			continue
		}

		urlLocator := page.Locator("div.group a").First()
		url, err := urlLocator.GetAttribute("href")
		if err != nil {
			slog.Error("error getting url at YC", slog.Any("error", err))
			url = ""
		}

		slog.Info("seed company Ycombinator",
			slog.String("CompanyName", name),
			slog.String("CompanyURL", url),
		)

		s.SeedCompany.ResultChan <- models.SeedCompanyResult{
			CompanyName: name,
			CompanyURL:  url,
		}

		if _, err := page.Goto(yc.URL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			slog.Error("error navigating back", slog.Any("error", err))
			break
		}

		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})

		count, _ = locator.Count()
		if count == 0 {
			slog.Warn("No nodes after re-navigation")
			break
		}
	}
}

func lastWord(text string) string {
	re := regexp.MustCompile(`\d+[hdwm]\s*ago`)
	text = re.ReplaceAllString(text, "")
	text = strings.TrimSpace(text)

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
