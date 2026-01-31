package service

import (
	"context"
	"os"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/repository"
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
		ResultChan: make(chan models.SeedCompanyResult, 500),
	}
}

func (s *SeedCompanyService) SeedCompanyConfigs(ctx context.Context, scraper *interfaces.ScraperClient) {
	logger.Info().Msg("seed company scraper started")
	defer close(s.SeedCompany.ResultChan)

	for i := 0; i < len(s.SeedCompany.Companies); i++ {
		select {
		case <-ctx.Done():
			logger.Info().Msg("SeedCompany stopping (context cancelled)")
			return
		default:
		}
		if s.SeedCompany.Companies[i].Name == "Peer list" {
			// s.SeedCompany.PWg.Add(1)
			// go func(sp models.SeedCompany) {
			// 	defer s.SeedCompany.PWg.Done()
			// 	s.GetSeedCompaniesFromPeerList(scraper, &sp, ctx)
			// }(s.SeedCompany.Companies[i])
			logger.Info().Msg("Skipping PeerList scraper as it's currently disabled")
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
	logger.Info().Msg("closing seedcompany waitgroups and result channel")
}

func (s *SeedCompanyService) GetSeedCompaniesFromPeerList(scraper *interfaces.ScraperClient, sp *models.SeedCompany, ctx context.Context) {
	logger.Info().Msg("worker started for peerlist")

	page, err := scraper.Browser.RunInNewTab()
	if err != nil {
		logger.Error().Err(err).Int("worker_id", -1).Msg("error creating page for peerlist")
		return
	}
	defer page.Close()

	logger.Info().Time("time", time.Now()).Msg("START processing for peerlist")

	if _, err := page.Goto(sp.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		logger.Error().Err(err).Int("worker_id", -1).Msg("error navigating to peerlist")
		return
	}

	locator := page.Locator(sp.Selector)
	count, err := locator.Count()
	if err != nil {
		logger.Error().Err(err).Int("worker_id", -1).Msg("error getting peerlist nodes count")
		return
	}

	logger.Info().Int("length", count).Str("selector", sp.Selector).Msg("Found nodes with selector in peerlist")

	go func() {
		for i := 0; i < count; i++ {
			logger.Info().Int("index", i).Msg("sending names to namesChan of peerlist from worker")

			item := locator.Nth(i)
			pElement := item.Locator("div:first-child > p")

			urlText, err := pElement.TextContent()
			if err != nil {
				logger.Error().Err(err).Msg("error getting text")
				continue
			}
			scraper.NamesChanClient.NamesChan <- LastWord(urlText)
		}
	}()

	s.UploadSeedCompanyToChannel(scraper)
}

func (s *SeedCompanyService) GetSeedCompaniesFromYCombinator(ctx context.Context, scraper *interfaces.ScraperClient, yc *models.SeedCompany) {
	logger.Info().Msg("worker started for ycombinator")

	 const maxCompanies = 10
	var processedCount atomic.Int32

	logger.Info().Msg("START processing for ycombinator")
	page, err := scraper.Browser.RunInNewTab()
	if err != nil {
		logger.Error().Err(err).Msg("error creating page for ycombinator")
		return
	}
	defer page.Close()
	logger.Info().Msg("START processing for ycombinator by running new tab")

	if _, err := page.Goto(yc.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		logger.Error().Err(err).Msg("error navigating to ycombinator")
		return
	}

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	locator := page.Locator(yc.Selector)
	count, _ := locator.Count()
	logger.Info().Int("count", count).Msg("Found companies")

	for i := 0; i < count; i++ {
		// Check if we've reached the limit
		if processedCount.Load() >= maxCompanies {
			logger.Info().Int("processed", int(processedCount.Load())).Msg("Reached maximum company limit, stopping YCombinator scraping")
			break
		}
		// if i >= count {
		// 	slog.Warn("Not enough nodes in Ycombinator", slog.Int("have", count), slog.Int("need", i))
		// 	break
		// }

		item := locator.Nth(i)

		nameLocator := item.Locator("span").First()
		name, err := nameLocator.TextContent()
		if err != nil {
			logger.Error().Err(err).Msg("error getting name at YC")
			continue
		}
		name = strings.TrimSpace(name)
		if err := item.Click(); err != nil {
			logger.Error().Err(err).Msg("error clicking item at YC")
			continue
		}

		urlLocator := page.Locator("div.group a").First()
		url, err := urlLocator.GetAttribute("href")
		if err != nil {
			logger.Error().Err(err).Msg("error getting url at YC")
			url = ""
		}

		logger.Info().Str("CompanyName", name).Str("CompanyURL", url).Msg("seed company Ycombinator")

		// Increment counter
		processedCount.Add(1)

		scrId := CreateSeedCompanyRepo(name, url, -1, *scraper)

		done := make(chan struct{})
		go func(companyUrl string, seedId uint, companyName string) {
			<-done
			scrapedJobResults, err := getJobResults(scraper.Browser, companyUrl)

			if err != nil {
				logger.Error().Str("company", companyName).Str("url", companyUrl).Err(err).Msg("FAILED to scrape jobs (likely no careers page)")
				return
			}
			repository.UpsertJob(scraper.DbClient.GetDB(), seedId, scrapedJobResults)

			logger.Info().Str("company", companyName).Int("job_count", len(scrapedJobResults)).Msg("SUCCESS: Upserted jobs")

		}(url, scrId, name)

		s.SeedCompany.ResultChan <- models.SeedCompanyResult{
			CompanyName:   name,
			CompanyURL:    url,
			SeedCompanyId: scrId,
		}
		time.Sleep(5 * time.Second)

		done <- struct{}{}

		//naviagte back to main page
		if _, err := page.Goto(yc.URL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			logger.Error().Err(err).Msg("error navigating back")
			break
		}

		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})

		count, _ = locator.Count()
		if count == 0 {
			logger.Warn().Msg("No nodes after re-navigation")
			break
		}
	}
}

func (s *SeedCompanyService) UploadSeedCompanyToChannel(scraper *interfaces.ScraperClient) {
	var searchWg sync.WaitGroup
	var searchEngineKey = os.Getenv("GOOGLE_SEARCH_ENGINE")

	const maxCompanies = 15
	 var processedCount atomic.Int32

	if searchEngineKey == "" {
		logger.Error().Msg("searchEngineKey is empty, skipping search")
		return
	}

	workerCount := 2
	for i := 0; i < workerCount; i++ {
		searchWg.Add(1)
		go func(workerID int) {
			defer searchWg.Done()
			logger.Info().Int("id", workerID).Msg("starting goroutine for search scraper in uploadSeedCompanyToChannel func by")
			for name := range scraper.NamesChanClient.NamesChan {
				// Check if we've reached the limit
				if processedCount.Load() >= maxCompanies {
					logger.Info().Int("worker", workerID).Int("processed", int(processedCount.Load())).Msg("Reached maximum company limit, stopping PeerList processing")
					return
				}
				if scraper.Search == nil {
					logger.Error().Msg("Search client is nil")
					continue
				}

				result, err := scraper.Search.SearchKeyWordInGoogle(
					name, workerID, searchEngineKey,
				)

				if err != nil {
					logger.Error().Err(err).Int("worker_id", workerID).Msg("error searching google")
					continue
				}

				if result == "" {
					logger.Warn().Str("name", name).Msg("empty result, skipping")
					continue
				}

				logger.Info().Str("url", result).Msg("company url in peerlist")

				// Increment counter
				 processedCount.Add(1)

				scrId := CreateSeedCompanyRepo(name, result, workerID, *scraper)

				done := make(chan struct{})
				go func(id uint, url string, companyName string) {
					<-done
					scrapedJobResults, err := getJobResults(scraper.Browser, url)
					if err != nil {
						logger.Error().Str("company", companyName).Str("url", url).Err(err).Msg("FAILED to scrape jobs (likely no careers page)")
						return
					}
					repository.UpsertJob(scraper.DbClient.GetDB(), id, scrapedJobResults)
					logger.Info().Str("company", companyName).Int("job_count", len(scrapedJobResults)).Msg("SUCCESS: Upserted jobs")
				}(scrId, result, name)

				time.Sleep(5 * time.Second)
				s.SeedCompany.ResultChan <- models.SeedCompanyResult{
					CompanyName:   name,
					CompanyURL:    result,
					SeedCompanyId: scrId,
				}
				done <- struct{}{}

			}
		}(i)
	}

	searchWg.Wait()
}

func CreateSeedCompanyRepo(name string, url string, workerID int, scraper interfaces.ScraperClient) uint {
	scr := repository.CreateSeedCompanyRepository(name, url)
	if err := repository.CreateSeedCompany(scr, scraper.DbClient.GetDB()); err != nil {
		logger.Error().Err(err).Int("worker_id", workerID).Msg("error creating seed company in DB")
	}
	return scr.ID
}

func getJobResults(browser interfaces.BrowserClient, companyUrl string) ([]models.LinkData, error) {
	return ScrapeJobs(browser, companyUrl)
}

func LastWord(text string) string {
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
