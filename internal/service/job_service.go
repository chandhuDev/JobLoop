package service

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/playwright-community/playwright-go"
)

type LinkData struct {
	URL  string
	Text string
}

var (
	socialDomains = map[string]bool{
		"facebook.com":  true,
		"twitter.com":   true,
		"x.com":         true,
		"linkedin.com":  true,
		"instagram.com": true,
		"youtube.com":   true,
		"tiktok.com":    true,
		"github.com":    true,
		"pinterest.com": true,
		"reddit.com":    true,
		"discord.com":   true,
		"medium.com":    true,
		"whatsapp.com":  true,
		"telegram.org":  true,
	}

	assetExtensions = []string{
		".png", ".jpg", ".jpeg", ".gif", ".svg", ".webp", ".ico",
		".pdf", ".doc", ".docx", ".zip", ".mp4", ".mp3",
		".css", ".js", ".woff", ".woff2", ".ttf",
	}

	noisePatterns = []string{
		"javascript:", "mailto:", "tel:", "sms:",
		"/cdn-cgi/", "/static/", "/assets/", "/images/", "/img/",
		"/fonts/", "/css/", "/js/", "/media/",
		"privacy", "terms", "cookie", "legal", "policy",
		"login", "signin", "signup", "register", "auth",
		"contact", "about-us", "blog", "news", "press",
		"faq", "help", "support",
	}

	excludeSelectors = []string{
		"header", "footer", "nav", "aside",
		"[role='navigation']", "[role='banner']", "[role='contentinfo']",
		".header", ".footer", ".nav", ".navbar", ".menu", ".sidebar", ".social",
		"#header", "#footer", "#nav", "#menu", "#sidebar",
		".cookie-banner", ".newsletter", ".subscribe",
		".social-links", ".social-icons",
	}
)

func ScrapeJobs(browser interfaces.BrowserClient, companyUrl string) ([]LinkData, error) {
	// var companyUrl = strings.TrimSpace("https://www.mux.com/")

	page, err := browser.RunInNewTab()
	if err != nil {
		slog.Error("Error creating new page", slog.String("of", companyUrl), slog.Any("is", err))
		return nil, err
	}
	defer page.Close()

	careersUrl, err := navigateToCareersPage(page, companyUrl)
	if err != nil {
		slog.Error("Error finding careers page", slog.String("of", companyUrl), slog.Any("is", err))
		return nil, err
	}

	err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State:   playwright.LoadStateNetworkidle,
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		slog.Warn("Timeout waiting for networkidle, continuing anyway", slog.String("url", careersUrl))
	}

	careersUrl = checkForJobsLink(page, careersUrl)

	page.WaitForTimeout(3000)

	slog.Info("Ready to scrape jobs", slog.String("url", careersUrl))

	removeNoisySections(page)

	links, err := extractAllLinks(page, careersUrl)
	if err != nil {
		slog.Error("Error extracting links", slog.Any("is", err))
		return nil, err
	}

	slog.Info("Extracted links after noise removal",
		slog.String("company", companyUrl),
		slog.Int("count", len(links)),
	)

	filteredLinks := filterLinks(links)

	slog.Info("Links after URL filtering",
		slog.String("company", companyUrl),
		slog.Int("count", len(filteredLinks)),
	)

	jobLinks := filterByJobScore(filteredLinks)

	slog.Info("Final job links",
		slog.String("company", companyUrl),
		slog.Int("count", len(jobLinks)),
	)

	return jobLinks, nil
}

// ==================== STAGE 1: DOM Surgery ====================
// Remove header, footer, nav, social sections BEFORE extracting links

func removeNoisySections(page playwright.Page) {
	for _, selector := range excludeSelectors {
		script := fmt.Sprintf(`
			document.querySelectorAll('%s').forEach(el => el.remove());
		`, selector)

		_, err := page.Evaluate(script)
		if err != nil {
			continue
		}
	}

	slog.Info("Removed noisy DOM sections")
}

func extractHiddenLinks(page playwright.Page) {
	jobCardSelectors := []string{
		"[class*='job-card']",
		"[class*='job-item']",
		"[class*='job-listing']",
		"[class*='position-card']",
		"[class*='opening-card']",
		"[class*='career-card']",
		"[class*='opportunity']",
		"[class*='posting']",
		".jobs-list > div",
		".jobs-list > li",
		".openings > div",
		".positions > div",
		"[data-job]",
		"[data-position]",
	}

	for _, selector := range jobCardSelectors {
		cards := page.Locator(selector)
		count, err := cards.Count()
		if err != nil || count == 0 {
			continue
		}

		slog.Info("Found job cards to hover", slog.String("selector", selector), slog.Int("count", count))

		for i := 0; i < count && i < 50; i++ {
			card := cards.Nth(i)
			err := card.Hover()
			if err != nil {
				continue
			}
			page.WaitForTimeout(200)
		}

		break
	}

	script := `
		// Make all hidden elements with href visible
		document.querySelectorAll('a[href]').forEach(a => {
			a.style.visibility = 'visible';
			a.style.opacity = '1';
			a.style.display = 'inline';
		});
		
		// Also check for elements that become links on hover
		document.querySelectorAll('[data-href], [data-url], [data-link]').forEach(el => {
			const href = el.getAttribute('data-href') || el.getAttribute('data-url') || el.getAttribute('data-link');
			if (href && !el.querySelector('a')) {
				const a = document.createElement('a');
				a.href = href;
				a.textContent = el.textContent || 'View Job';
				el.appendChild(a);
			}
		});
	`
	page.Evaluate(script)
}

func checkForJobsLink(page playwright.Page, currentUrl string) string {
	viewJobsTexts := []string{
		"View open roles",
		"View all jobs",
		"View jobs",
		"View openings",
		"View positions",
		"See all jobs",
		"See open roles",
		"See openings",
		"Explore open roles",
		"Explore our open roles",
		"Explore jobs",
		"Browse jobs",
		"Browse openings",
		"Open positions",
		"Open roles",
		"Current openings",
		"Job openings",
		"All jobs",
		"See opportunities",
	}

	for _, text := range viewJobsTexts {
		locator := page.Locator(fmt.Sprintf(`a:has-text("%s")`, text)).First()

		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		href, err := locator.GetAttribute("href")
		if err != nil || href == "" {
			slog.Info("Clicking 'view jobs' link (no href)", slog.String("text", text))
			err = locator.Click()
			if err == nil {
				page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
					State: playwright.LoadStateNetworkidle,
				})
				newUrl := page.URL()
				slog.Info("Clicked 'view jobs' link",
					slog.String("text", text),
					slog.String("newUrl", newUrl),
				)
				return newUrl
			}
			continue
		}

		resolved := resolveUrl(currentUrl, href)
		if resolved == "" {
			continue
		}

		slog.Info("Navigating to 'view jobs' link",
			slog.String("text", text),
			slog.String("href", href),
			slog.String("resolved", resolved),
		)

		resp, err := page.Goto(resolved, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(20000),
		})

		actualUrl := page.URL()

		if err != nil {
			slog.Warn("Navigation had error but checking actual URL",
				slog.String("error", err.Error()),
				slog.String("actualUrl", actualUrl),
			)
			if actualUrl != currentUrl && actualUrl != "" {
				slog.Info("Landed on new page despite error", slog.String("url", actualUrl))
				return actualUrl
			}
			continue
		}

		if resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
			slog.Info("Found and navigated to jobs page",
				slog.String("text", text),
				slog.String("url", actualUrl),
			)
			return actualUrl
		}
	}

	hrefPatterns := []string{
		`a[href*="open-positions"]`,
		`a[href*="openings"]`,
		`a[href*="all-jobs"]`,
		`a[href*="job-listing"]`,
		`a[href*="positions"]`,
	}

	for _, selector := range hrefPatterns {
		locator := page.Locator(selector).First()

		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		href, err := locator.GetAttribute("href")
		if err != nil || href == "" {
			continue
		}

		resolved := resolveUrl(currentUrl, href)
		if resolved == "" {
			continue
		}

		slog.Info("Navigating to jobs page via href pattern",
			slog.String("selector", selector),
			slog.String("resolved", resolved),
		)

		resp, err := page.Goto(resolved, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(20000),
		})

		actualUrl := page.URL()

		if err != nil {
			slog.Warn("Navigation had error but checking actual URL",
				slog.String("error", err.Error()),
				slog.String("actualUrl", actualUrl),
			)
			if actualUrl != currentUrl && actualUrl != "" {
				slog.Info("Landed on new page despite error", slog.String("url", actualUrl))
				return actualUrl
			}
			continue
		}

		if resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
			slog.Info("Found jobs page via href pattern",
				slog.String("selector", selector),
				slog.String("url", actualUrl),
			)
			return actualUrl
		}
	}

	return currentUrl
}

func extractAllLinks(page playwright.Page, baseUrl string) ([]LinkData, error) {
	extractHiddenLinks(page)

	script := `
		Array.from(document.querySelectorAll('a[href]')).map(a => ({
			href: a.getAttribute('href') || '',
			text: a.innerText.trim() || a.getAttribute('aria-label') || a.getAttribute('title') || ''
		})).filter(link => link.href !== '')
	`

	result, err := page.Evaluate(script)
	if err != nil {
		return nil, fmt.Errorf("failed to extract links: %w", err)
	}

	links := []LinkData{}
	seen := make(map[string]bool)

	if items, ok := result.([]interface{}); ok {
		for _, item := range items {
			if m, ok := item.(map[string]interface{}); ok {
				href, _ := m["href"].(string)
				text, _ := m["text"].(string)

				resolved := resolveUrl(baseUrl, href)
				if resolved == "" {
					continue
				}

				if seen[resolved] {
					continue
				}

				cleanedText := cleanText(text)
				if cleanedText == "" {
					continue
				}

				seen[resolved] = true
				links = append(links, LinkData{
					URL:  resolved,
					Text: cleanedText,
				})
			}
		}
	}

	iframeLinks := extractLinksFromIframes(page, baseUrl)
	for _, link := range iframeLinks {
		if !seen[link.URL] {
			seen[link.URL] = true
			links = append(links, link)
		}
	}

	return links, nil
}

func extractLinksFromIframes(page playwright.Page, baseUrl string) []LinkData {
	links := []LinkData{}

	iframes := page.Locator("iframe")
	count, err := iframes.Count()
	if err != nil || count == 0 {
		return links
	}

	slog.Info("Found iframes", slog.Int("count", count))

	for i := 0; i < count; i++ {
		iframe := iframes.Nth(i)

		src, err := iframe.GetAttribute("src")
		if err != nil || src == "" {
			continue
		}

		if !isJobBoardIframe(src) {
			continue
		}

		slog.Info("Processing job board iframe", slog.String("src", src))

		frameLocator := iframe.ContentFrame()

		anchorLocator := frameLocator.Locator("a[href]")
		anchorCount, err := anchorLocator.Count()
		if err != nil || anchorCount == 0 {
			slog.Info("No links found in iframe", slog.String("src", src))
			continue
		}

		slog.Info("Found links in iframe", slog.Int("count", anchorCount), slog.String("src", src))

		for j := 0; j < anchorCount; j++ {
			anchor := anchorLocator.Nth(j)

			href, err := anchor.GetAttribute("href")
			if err != nil || href == "" {
				continue
			}

			text, _ := anchor.InnerText()
			if text == "" {
				text, _ = anchor.GetAttribute("aria-label")
			}
			if text == "" {
				text, _ = anchor.GetAttribute("title")
			}

			resolved := resolveUrl(src, href)
			if resolved == "" {
				continue
			}

			cleanedText := cleanText(text)
			if cleanedText == "" {
				continue
			}

			links = append(links, LinkData{
				URL:  resolved,
				Text: cleanedText,
			})
		}
	}

	return links
}

func isJobBoardIframe(src string) bool {
	jobBoards := []string{
		"greenhouse.io",
		"lever.co",
		"workday.com",
		"ashbyhq.com",
		"bamboohr.com",
		"recruitee.com",
		"workable.com",
		"smartrecruiters.com",
		"icims.com",
		"jobvite.com",
		"myworkdayjobs.com",
	}

	lower := strings.ToLower(src)
	for _, board := range jobBoards {
		if strings.Contains(lower, board) {
			return true
		}
	}

	return false
}

func filterLinks(links []LinkData) []LinkData {
	filtered := []LinkData{}

	for _, link := range links {
		if passesURLFilters(link.URL) {
			filtered = append(filtered, link)
		}
	}

	return filtered
}

func passesURLFilters(link string) bool {
	lower := strings.ToLower(link)

	for _, pattern := range noisePatterns {
		if strings.Contains(lower, pattern) {
			return false
		}
	}

	for _, ext := range assetExtensions {
		if strings.HasSuffix(lower, ext) {
			return false
		}
	}

	parsed, err := url.Parse(link)
	if err != nil {
		return false
	}

	host := strings.ToLower(parsed.Host)
	host = strings.TrimPrefix(host, "www.")

	for domain := range socialDomains {
		if host == domain || strings.HasSuffix(host, "."+domain) {
			return false
		}
	}

	if strings.HasPrefix(link, "#") {
		return false
	}

	return true
}

func filterByJobScore(links []LinkData) []LinkData {
	jobLinks := []LinkData{}

	for _, link := range links {
		score := calculateJobScore(link.URL, link.Text)
		if score >= 2 {
			slog.Info("Job Link Found", slog.String("url", link.URL), slog.String("text", link.Text))
			jobLinks = append(jobLinks, link)
		}
	}

	return jobLinks
}

func calculateJobScore(linkUrl string, text string) int {
	combined := strings.ToLower(linkUrl + " " + text)
	score := 0

	jobURLPatterns := []string{
		"/job/", "/jobs/", "/career/", "/careers/",
		"/position/", "/positions/", "/opening/", "/openings/",
		"/role/", "/roles/", "/vacancy/", "/vacancies/",
		"/apply/", "/opportunity/", "/opportunities/",
		"greenhouse.io", "lever.co", "workday.com", "ashbyhq.com",
		"boards.io", "bamboohr.com", "recruitee.com", "workable.com",
		"smartrecruiters.com", "icims.com", "jobvite.com",
	}

	for _, pattern := range jobURLPatterns {
		if strings.Contains(combined, pattern) {
			score += 3
		}
	}

	jobTitleKeywords := []string{
		"engineer", "developer", "designer", "manager", "director",
		"analyst", "specialist", "coordinator", "lead", "senior",
		"junior", "intern", "associate", "consultant", "architect",
		"scientist", "researcher", "product", "marketing", "sales",
		"remote", "full-time", "part-time", "contract", "hybrid",
	}

	for _, keyword := range jobTitleKeywords {
		if strings.Contains(combined, keyword) {
			score += 1
		}
	}

	negativePatterns := []string{
		"blog", "article", "news", "press", "team", "about",
		"culture", "values", "benefits", "perks",
	}

	for _, pattern := range negativePatterns {
		if strings.Contains(combined, pattern) {
			score -= 1
		}
	}

	return score
}

func cleanText(s string) string {
	fields := strings.Fields(s)
	return strings.Join(fields, " ")
}

func navigateToCareersPage(page playwright.Page, companyUrl string) (string, error) {
	companyUrl = strings.TrimRight(companyUrl, "/")

	careerPaths := []string{
		"/careers",
		"/jobs",
		"/company/jobs",
		"/company/careers",
		"/about/careers",
		"/en/careers",
		"/work-with-us",
	}

	for _, path := range careerPaths {
		targetUrl := companyUrl + path

		resp, err := page.Goto(targetUrl, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			Timeout:   playwright.Float(15000),
		})

		if err != nil {
			continue
		}

		if resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
			actualUrl := page.URL()
			slog.Info("Found careers page via direct URL",
				slog.String("requested", targetUrl),
				slog.String("actual", actualUrl),
			)
			return actualUrl, nil
		}
	}

	slog.Info("Direct URLs failed, looking for careers link on homepage", slog.String("company", companyUrl))

	_, err := page.Goto(companyUrl, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return "", fmt.Errorf("failed to load homepage: %w", err)
	}

	careersTexts := []string{
		"Careers", "careers", "Jobs", "jobs",
		"Join Us", "Join us", "Work with us", "Work With Us",
		"We're Hiring", "We're hiring",
	}

	for _, text := range careersTexts {
		locator := page.Locator(fmt.Sprintf(`text="%s"`, text)).First()

		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		href, err := locator.GetAttribute("href")
		if err == nil && href != "" {
			resolved := resolveUrl(companyUrl, href)
			if resolved != "" {
				resp, err := page.Goto(resolved, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
					Timeout:   playwright.Float(15000),
				})
				if err == nil && resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
					actualUrl := page.URL()
					slog.Info("Found careers link on element",
						slog.String("href", resolved),
						slog.String("actual", actualUrl),
					)
					return actualUrl, nil
				}
			}
		}

		ancestorAnchor := locator.Locator("xpath=ancestor::a").First()
		count, err = ancestorAnchor.Count()
		if err == nil && count > 0 {
			href, err = ancestorAnchor.GetAttribute("href")
			if err == nil && href != "" {
				resolved := resolveUrl(companyUrl, href)
				if resolved != "" {
					resp, err := page.Goto(resolved, playwright.PageGotoOptions{
						WaitUntil: playwright.WaitUntilStateDomcontentloaded,
						Timeout:   playwright.Float(15000),
					})
					if err == nil && resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
						actualUrl := page.URL()
						slog.Info("Found careers link via ancestor anchor",
							slog.String("href", resolved),
							slog.String("actual", actualUrl),
						)
						return actualUrl, nil
					}
				}
			}
		}

		childAnchor := locator.Locator("a").First()
		count, err = childAnchor.Count()
		if err == nil && count > 0 {
			href, err = childAnchor.GetAttribute("href")
			if err == nil && href != "" {
				resolved := resolveUrl(companyUrl, href)
				if resolved != "" {
					resp, err := page.Goto(resolved, playwright.PageGotoOptions{
						WaitUntil: playwright.WaitUntilStateDomcontentloaded,
						Timeout:   playwright.Float(15000),
					})
					if err == nil && resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
						actualUrl := page.URL()
						slog.Info("Found careers link via child anchor",
							slog.String("href", resolved),
							slog.String("actual", actualUrl),
						)
						return actualUrl, nil
					}
				}
			}
		}
	}

	hrefSelectors := []string{
		`a[href*="careers"]`,
		`a[href*="jobs"]`,
		`a[href*="career"]`,
		`a[href*="job"]`,
	}

	page.Goto(companyUrl, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(15000),
	})

	for _, selector := range hrefSelectors {
		locator := page.Locator(selector).First()

		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		href, err := locator.GetAttribute("href")
		if err == nil && href != "" {
			resolved := resolveUrl(companyUrl, href)
			if resolved != "" {
				resp, err := page.Goto(resolved, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
					Timeout:   playwright.Float(15000),
				})
				if err == nil && resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
					actualUrl := page.URL()
					slog.Info("Found careers link via href selector",
						slog.String("href", resolved),
						slog.String("actual", actualUrl),
					)
					return actualUrl, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no careers page found for %s", companyUrl)
}

func resolveUrl(baseUrl, href string) string {
	if strings.HasPrefix(href, "javascript:") {
		extractedUrl := extractUrlFromJavascript(href)
		if extractedUrl != "" {
			return resolveUrl(baseUrl, extractedUrl)
		}
		return ""
	}

	if href == "" || href == "#" {
		return ""
	}

	if strings.HasPrefix(href, "http") {
		return href
	}

	if strings.HasPrefix(href, "//") {
		return "https:" + href
	}

	parsed, err := url.Parse(baseUrl)
	if err != nil {
		return ""
	}

	origin := parsed.Scheme + "://" + parsed.Host

	if strings.HasPrefix(href, "/") {
		return origin + href
	}

	baseUrl = strings.TrimRight(baseUrl, "/")
	return baseUrl + "/" + href
}

func extractUrlFromJavascript(js string) string {
	patterns := []string{
		`window\.open\(['"]([^'"]+)['"]`,
		`window\.location\s*=\s*['"]([^'"]+)['"]`,
		`location\.href\s*=\s*['"]([^'"]+)['"]`,
	}

	for _, pattern := range patterns {
		re := regexp.MustCompile(pattern)
		matches := re.FindStringSubmatch(js)
		if len(matches) > 1 {
			decoded, err := url.QueryUnescape(matches[1])
			if err != nil {
				return matches[1]
			}
			return decoded
		}
	}

	return ""
}
