package service

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"
	"sync"

	"time"
	"strconv"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/repository"
	"github.com/playwright-community/playwright-go"
)

type SeedCompanyService struct {
	SeedCompany *models.SeedCompanyArray
}

type CompanyData struct {
	Name      string // Company name from YC listing
	YCUrl     string // YCombinator profile URL (/companies/doordash)
	ActualURL string // Actual company website URL (scraped from YC profile)
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

	// Step 1: Scrape all company names and YC URLs
	companyData := scrapeCompaniesWithScroll(page, yc.Selector)
	logger.Info().Int("total_companies", len(companyData)).Msg("Finished scraping all companies")

	// Step 2: Visit each YC profile to get actual company URLs
	companyData, err = getCompanyUrls(page, companyData)
	if err != nil {
		logger.Error().Err(err).Msg("error at getCompanyUrls")
		return
	}

	// Step 3: Process each company
	for i, company := range companyData {
		logger.Info().Int("index", i+1).Int("total", len(companyData)).Str("company", company.Name).Str("url", company.ActualURL).Msg("Processing company")

		// Skip if no actual URL found
		if company.ActualURL == "" {
			logger.Warn().Str("company", company.Name).Msg("Skipping company - no actual URL found")
			continue
		}

		scrId := CreateSeedCompanyRepo(company.Name, company.ActualURL, -1, *scraper)

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

		}(company.ActualURL, scrId, company.Name)

		s.SeedCompany.ResultChan <- models.SeedCompanyResult{
			CompanyName:   company.Name,
			CompanyURL:    company.ActualURL,
			SeedCompanyId: scrId,
		}

		done <- struct{}{}

		time.Sleep(3 * time.Second)
	}
}

func scrapeCompaniesWithScroll(page playwright.Page, selector string) []CompanyData {
	var allCompanies []CompanyData
	seenNames := make(map[string]bool)
	noNewCompaniesSince := 0
	maxNoNewAttempts := 3
	previousVisibleCount := 0
	maxlengthEnv := os.Getenv("MAX_LEN")

	maxlength, _ := strconv.Atoi(maxlengthEnv)
    logger.Info().Msg("Starting company scraping with scroll")

	for {
		// Use JavaScript to extract all visible companies at once
		result, err := page.Evaluate(fmt.Sprintf(`
			() => {
				const items = document.querySelectorAll('%s');
				const companies = [];
				
				items.forEach((item, index) => {
					const span = item.querySelector('span');
					const name = span ? span.textContent.trim() : '';
					const href = item.getAttribute('href') || '';
					
					if (name) {
						companies.push({ name, href, index });
					}
				});
				
				return companies;
			}
		`, selector))

		if err != nil {
			logger.Error().Err(err).Msg("Failed to extract companies with JS")
			break
		}

		// Parse the result
		companiesArray, ok := result.([]interface{})
		if !ok {
			logger.Warn().Msg("Unexpected result format from JS")
			break
		}

		currentVisibleCount := len(companiesArray)
		logger.Info().Int("visible_count", currentVisibleCount).Int("previous_count", previousVisibleCount).Msg("Found visible company anchors")

		newCompaniesFound := 0
		for _, item := range companiesArray {
			companyMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			name, _ := companyMap["name"].(string)
			href, _ := companyMap["href"].(string)
			index, _ := companyMap["index"].(float64)

			name = strings.TrimSpace(name)

			// Skip if empty or already seen
			if name == "" || seenNames[name] {
				continue
			}

			fullURL := href
			if strings.HasPrefix(href, "/") {
				pageURL := page.URL()
				baseURL, _ := url.Parse(pageURL)
				if baseURL != nil {
					relURL, _ := url.Parse(href)
					fullURL = baseURL.ResolveReference(relURL).String()
				}
			}

			// Add to results with YCUrl populated
			allCompanies = append(allCompanies, CompanyData{
				Name:      name,
				YCUrl:     fullURL,
				ActualURL: "",
			})
			seenNames[name] = true
			newCompaniesFound++

			logger.Info().Int("scraped_count", len(allCompanies)).Int("index", int(index)).Str("name", name).Str("yc_url", fullURL).Msg("SCRAPED company")
		}

		logger.Info().Int("new_found", newCompaniesFound).Int("total_scraped", len(allCompanies)).Msg("Scraping batch complete")

		if currentVisibleCount == previousVisibleCount {
			noNewCompaniesSince++
			logger.Warn().Int("attempts", noNewCompaniesSince).Int("stuck_at_count", currentVisibleCount).Msg("Visible count did not increase after scroll")

			if noNewCompaniesSince >= maxNoNewAttempts {
				logger.Info().Msg("No new companies loading after multiple scroll attempts, stopping")
				break
			}
		} else {
			noNewCompaniesSince = 0
			logger.Info().Int("old_count", previousVisibleCount).Int("new_count", currentVisibleCount).Int("loaded", currentVisibleCount-previousVisibleCount).Msg("New companies loaded after scroll")
		}

		previousVisibleCount = currentVisibleCount

		logger.Info().Msg("Scrolling to load more companies...")

		lastElementScrolled, _ := page.Evaluate(fmt.Sprintf(`
			() => {
				const items = document.querySelectorAll('%s');
				if (items.length > 0) {
					const lastItem = items[items.length - 1];
					lastItem.scrollIntoView({ behavior: 'smooth', block: 'end' });
					return true;
				}
				return false;
			}
		`, selector))

		logger.Info().Bool("scrolled_to_last", lastElementScrolled == true).Msg("Scrolled to last element")
		time.Sleep(1500 * time.Millisecond)

		page.Evaluate(`() => window.scrollTo(0, document.body.scrollHeight)`)
		time.Sleep(1500 * time.Millisecond)

		page.Evaluate(`() => window.scrollBy(0, 1000)`)
		time.Sleep(1500 * time.Millisecond)

		if len(allCompanies) > maxlength {
			logger.Warn().Msg("Reached safety limit of 5000 companies")
			break
		}
	}

	logger.Info().Int("total_scraped", len(allCompanies)).Msg("Scraping complete")
	return allCompanies
}

func getCompanyUrls(page playwright.Page, data []CompanyData) ([]CompanyData, error) {
	logger.Info().Int("total", len(data)).Msg("Starting to fetch actual company URLs")

	for i := range data {
		logger.Info().Int("index", i+1).Int("total", len(data)).Str("company", data[i].Name).Str("yc_url", data[i].YCUrl).Msg("Fetching actual URL")

		_, err := page.Goto(data[i].YCUrl, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(30000),
		})
		if err != nil {
			logger.Error().Err(err).Str("company", data[i].Name).Str("yc_url", data[i].YCUrl).Msg("Failed to navigate to YC profile")
			continue
		}

		page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})

		urlLocator := page.Locator("div.group a").First()
		actualURL, err := urlLocator.GetAttribute("href", playwright.LocatorGetAttributeOptions{
			Timeout: playwright.Float(5000),
		})
		if err != nil {
			logger.Warn().Err(err).Str("company", data[i].Name).Msg("Failed to get actual company URL")
			actualURL = ""
		}

		// Update the ActualURL in the same struct
		data[i].ActualURL = actualURL

		logger.Info().Str("company", data[i].Name).Str("actual_url", actualURL).Msg("SCRAPED actual company URL")

		// Small delay to avoid overwhelming the server
		time.Sleep(500 * time.Millisecond)
	}

	logger.Info().Int("total_with_urls", len(data)).Msg("Finished fetching actual URLs")
	return data, nil
}

func (s *SeedCompanyService) UploadSeedCompanyToChannel(scraper *interfaces.ScraperClient) {
	var searchWg sync.WaitGroup

	// const maxCompanies = 15
	// var processedCount atomic.Int32

	workerCount := 2
	for i := 0; i < workerCount; i++ {
		searchWg.Add(1)
		go func(workerID int) {
			defer searchWg.Done()
			logger.Info().Int("id", workerID).Msg("starting goroutine for search scraper in uploadSeedCompanyToChannel func by")
			for name := range scraper.NamesChanClient.NamesChan {
				// Check if we've reached the limit
				// if processedCount.Load() >= maxCompanies {
				// 	logger.Info().Int("worker", workerID).Int("processed", int(processedCount.Load())).Msg("Reached maximum company limit, stopping PeerList processing")
				// 	return
				// }
				if scraper.Search == nil {
					logger.Error().Msg("Search client is nil")
					continue
				}

				result, err := scraper.Search.SearchKeyword(
					name, workerID,
				)

				if err != nil {
					logger.Error().Err(err).Int("worker_id", workerID).Msg("error searching google")
					continue
				}

				if result == "" {
					logger.Warn().Str("name", name).Msg("empty result, skipping")
					continue
				}

				logger.Info().Str("url", result).Msg("company url result")

				// Increment counter
				// processedCount.Add(1)

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
