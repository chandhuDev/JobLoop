package service

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"

	"github.com/chandhuDev/JobLoop/internal/interfaces"
	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/internal/models"
)

/* ================= CONFIG ================= */

var (
	careerKeywords = []string{"careers", "jobs"}

	ctaKeywords = []string{
		"view all", "view jobs", "view openings", "browse jobs", "browse openings", "view roles", "view positions", "open positions", "see openings", "see jobs", "search jobs", "explore open roles", "explore jobs", "join us", "see all jobs", "see our open roles", "explore roles", "view job positions", "view open postions", "see open jobs", "find a job",
	}

	jobKeywords = []string{
		"senior", "lead", "engineer", "experience", "sales",
		"platform", "head", "product", "manager", "principal",
		"strategy", "solutions", "partner", "employee", "finance",
		"account", "executive", "consultant", "sr.", "strategic",
		"operations", "acquisition", "business", "analyst", "customer",
	}

	jobWaitTimeout = 3 * time.Second
)

/* ================= MAIN ================= */

func ScrapeJobs(browser interfaces.BrowserClient, companyURL string) ([]models.LinkData, error) {
	if browser == nil {
		return nil, fmt.Errorf("browser is nil")
	}

	page, err := browser.RunInNewTab()
	if err != nil {
		return nil, fmt.Errorf("failed to create new tab: %w", err)
	}
	if page == nil {
		return nil, fmt.Errorf("page is nil")
	}
	defer page.Close()

	baseURL, err := url.Parse(companyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid company URL: %w", err)
	}

	logger.Info().Str("homepage", companyURL).Msg("Homepage")

	resp, err := page.Goto(companyURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to homepage: %w", err)
	}

	if resp != nil {
		logger.Info().Int("status", resp.Status()).Str("url", resp.URL()).Msg("Homepage response")
	}

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	/* ---------- FIND CAREERS PAGE ---------- */

	careersURL, _ := findCareersLink(page, baseURL)

	// Fallback: try common career paths if no link found
	if careersURL == "" {
		logger.Info().Msg("No careers link found, trying common paths")
		careersURL = tryCommonCareerPaths(page, baseURL)
	}

	if careersURL == "" {
		return nil, fmt.Errorf("no careers/jobs page found")
	}

	logger.Info().Str("careers_page", careersURL).Msg("Careers page")

	resp, err = page.Goto(careersURL, playwright.PageGotoOptions{
		Timeout: playwright.Float(30000),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to navigate to careers page: %w", err)
	}

	if resp != nil {
		logger.Info().Int("status", resp.Status()).Str("url", resp.URL()).Msg("Careers page response")
		if resp.Status() >= 400 {
			return nil, fmt.Errorf("careers page returned status %d", resp.Status())
		}
	}

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	/* ---------- FIND CTA LINKS FIRST ---------- */

	currentURL := page.URL()
	currentBaseURL, err := url.Parse(currentURL)
	if err != nil {
		currentBaseURL = baseURL
	}
	logger.Info().Str("current_url", currentURL).Msg("Current page URL after careers navigation")

	ctas := findCTAs(page)
	logger.Info().Int("cta_count", len(ctas)).Msg("Found CTA buttons")

	/* ---------- HANDLE EACH CTA ---------- */

	for _, cta := range ctas {
		logger.Info().Str("cta_text", cta.Text).Str("cta_href", cta.RawHref).Msg("CTA")

		if strings.HasPrefix(cta.RawHref, "#") {
			logger.Info().Msg("Hash CTA detected - scraping current page only")

			jobs := scanForJobsWithPagination(page, currentBaseURL, 10)
			jobs = dedupeJobs(jobs)

			if len(jobs) > 0 {
				logger.Info().Msg("Jobs found on careers page (hash CTA)")
				logJobs(jobs)
				return jobs, nil
			}

			continue
		}

		target := resolveURL(cta.RawHref, currentBaseURL)
		logger.Info().Str("cta_url", target).Msg("Navigating CTA URL")

		_, err = page.Goto(target, playwright.PageGotoOptions{
			Timeout: playwright.Float(30000),
		})
		if err != nil || (resp != nil && resp.Status() >= 400) {
			if currentBaseURL.Host != baseURL.Host {
				fallbackTarget := resolveURL(cta.RawHref, baseURL)
				logger.Info().Str("failed_url", target).Str("fallback_url", fallbackTarget).Msg("First URL failed, trying with original base URL")

				resp, err = page.Goto(fallbackTarget, playwright.PageGotoOptions{
					Timeout: playwright.Float(30000),
				})

				if err != nil {
					logger.Warn().Str("url", fallbackTarget).Err(err).Msg("Fallback URL also failed")
					continue
				}

				if resp != nil && resp.Status() >= 400 {
					logger.Warn().Str("url", fallbackTarget).Int("status", resp.Status()).Msg("Fallback URL returned error status")
					continue
				}

				target = fallbackTarget
			} else {
				logger.Warn().Str("url", target).Err(err).Msg("Failed to navigate to CTA")
				continue
			}
		}

		jobPageURL := page.URL()
		jobBaseURL, _ := url.Parse(jobPageURL)
		if jobBaseURL == nil {
			jobBaseURL = currentBaseURL
		}

		waitForJobContent(page)

		jobs := scanForJobsWithPagination(page, jobBaseURL, 10)
		jobs = dedupeJobs(jobs)

		if len(jobs) > 0 {
			logger.Info().Msg("Jobs found via CTA")
			logJobs(jobs)
			return jobs, nil
		}
	}

	/* ---------- FALLBACK: DIRECT SCAN IF NO CTAs ---------- */

	if len(ctas) == 0 {
		logger.Info().Msg("No CTAs found, scanning careers page directly")

		jobs := scanForJobsWithPagination(page, currentBaseURL, 10)
		jobs = dedupeJobs(jobs)

		if len(jobs) > 0 {
			logger.Info().Msg("Jobs found directly on careers page")
			logJobs(jobs)
			return jobs, nil
		}
	}

	logger.Info().Msg("No jobs found")
	return nil, nil
}

func tryCommonCareerPaths(page playwright.Page, baseURL *url.URL) string {
	commonPaths := []string{
		"/careers",
		"/jobs",
		"/careers/",
		"/jobs/",
		"/en/careers",
		"/about/careers",
		"/company/careers",
	}

	for _, path := range commonPaths {
		testURL := baseURL.Scheme + "://" + baseURL.Host + path
		logger.Info().Str("url", testURL).Msg("Trying career path")

		resp, err := page.Goto(testURL, playwright.PageGotoOptions{
			Timeout: playwright.Float(10000),
		})
		if err != nil {
			continue
		}

		if resp != nil && resp.Status() >= 200 && resp.Status() < 400 {
			logger.Info().Str("url", testURL).Int("status", resp.Status()).Msg("Found valid career path")
			return testURL
		}
	}

	return ""
}

/* ================= FIND CAREERS ================= */

func findCareersLink(page playwright.Page, base *url.URL) (string, error) {
	js := `
	() => {
		const K = %s;
		for (const a of document.querySelectorAll("footer a, a")) {
			const t = a.innerText?.toLowerCase();
			if (t && K.some(k => t.includes(k))) {
				return a.getAttribute("href");
			}
		}
		return null;
	}
	`

	res, _ := page.Evaluate(fmt.Sprintf(js, toJSArray(careerKeywords)))
	if res == nil {
		return "", nil
	}

	href, ok := res.(string)
	if !ok {
		return "", nil
	}

	return resolveURL(href, base), nil
}

/* ================= FIND CTAs ================= */

type CTA struct {
	Text    string
	RawHref string
	XPath   string
}

func findCTAs(page playwright.Page) []CTA {
	js := `
	() => {
		const K = %s;

		function xpath(el) {
			const parts = [];
			while (el && el.nodeType === 1) {
				let i = 1;
				for (let sib = el.previousSibling; sib; sib = sib.previousSibling)
					if (sib.nodeType === 1 && sib.tagName === el.tagName) i++;
				parts.unshift(el.tagName.toLowerCase() + "[" + i + "]");
				el = el.parentNode;
			}
			return "/" + parts.join("/");
		}

		const out = [];
		for (const a of document.querySelectorAll("a")) {
			if (a.closest("header") || a.closest("footer") || a.closest("nav")) continue;

			const t = a.innerText?.trim()?.toLowerCase();
			if (!t || !K.some(k => t.includes(k))) continue;

			const href = a.getAttribute("href");
			if (!href || href === "#" || href.startsWith("javascript")) continue;

			out.push({
				text: a.innerText.trim(),
				href: href,
				xpath: xpath(a)
			});
		}
		return out;
	}
	`

	res, _ := page.Evaluate(fmt.Sprintf(js, toJSArray(ctaKeywords)))
	var ctas []CTA

	arr, ok := res.([]interface{})
	if !ok {
		return ctas
	}

	for _, v := range arr {
		m := v.(map[string]interface{})
		ctas = append(ctas, CTA{
			Text:    toString(m["text"]),
			RawHref: toString(m["href"]),
			XPath:   toString(m["xpath"]),
		})
	}

	return ctas
}

/* ================= JOB SCAN ================= */

func scanForJobs(page playwright.Page, baseURL *url.URL) []models.LinkData {
	logger.Info().Msg("ENTER scanForJobs (container-aware, text-only)")

	page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
		State: playwright.LoadStateNetworkidle,
	})

	var jobs []models.LinkData
	seen := make(map[string]bool)

	// Try to find Greenhouse/Lever/ATS iframe
	atsIframe := page.FrameLocator("iframe[src*='greenhouse.io'], iframe[src*='lever.co'], iframe[src*='workday.com'], iframe[src*='ashbyhq.com']")

	// Check if iframe exists by trying to get an anchor inside it
	anchors := atsIframe.Locator("a[href]")
	count, err := anchors.Count()

	if err != nil || count == 0 {
		logger.Info().Msg("No ATS iframe found, falling back to main page")
		anchors = page.Locator("a[href]")
		count, _ = anchors.Count()
	} else {
		logger.Info().Int("anchor_count", count).Msg("Found ATS iframe, scanning inside")
	}

	for i := 0; i < count; i++ {
		a := anchors.Nth(i)

		href, err := a.GetAttribute("href")
		if err != nil {
			continue
		}
		href = strings.TrimSpace(href)
		if href == "" || href == "#" {
			continue
		}
		absoluteURL := resolveURL(href, baseURL)
		if isPaginationOrFilterURL(absoluteURL) {
			logger.Debug().Str("href", absoluteURL).Msg("Skipping pagination/filter URL")
			continue
		}
		logger.Debug().Str("href", href).Str("absolute", absoluteURL).Msg("Processing anchor")

		text, _ := a.TextContent()
		text = strings.TrimSpace(text)

		if len(text) < 3 {
			container := a.Locator("xpath=ancestor::tr[1] | ancestor::li[1] | ancestor::div[1]")
			if cCount, _ := container.Count(); cCount > 0 {
				ct, _ := container.First().TextContent()
				text = strings.TrimSpace(ct)
			}
		}

		lowerText := strings.ToLower(text)

		if !containsAny(lowerText, jobKeywords) {
			continue
		}

		// NEW: Additional filter - if text is too long, it's likely a container/filter section
		if len(text) > 500 {
			logger.Debug().Str("href", absoluteURL).Int("text_length", len(text)).Msg("Skipping - text too long (likely container)")
			continue
		}

		if seen[absoluteURL] {
			continue
		}
		seen[absoluteURL] = true

		logger.Info().Str("href", absoluteURL).Str("text", text).Msg("Job found")

		jobs = append(jobs, models.LinkData{
			Text: text,
			URL:  absoluteURL,
		})
	}

	logger.Info().Int("jobs", len(jobs)).Msg("Jobs found after scan")
	return jobs
}

func isPaginationOrFilterURL(urlStr string) bool {
	parsed, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	// Check for pagination query params
	query := parsed.Query()
	paginationParams := []string{"page", "p", "spage", "paged", "_page", "pageNum", "pageNumber", "offset", "start"}
	for _, param := range paginationParams {
		if query.Get(param) != "" {
			return true
		}
	}

	// Check for hash fragments that indicate filters/results sections
	if parsed.Fragment != "" {
		filterHashes := []string{"results", "filter", "filters", "search", "page"}
		fragment := strings.ToLower(parsed.Fragment)
		for _, filterHash := range filterHashes {
			if strings.Contains(fragment, filterHash) {
				return true
			}
		}
	}

	return false
}

type PaginationPattern struct {
	ParamName          string
	BaseURL            string
	StartIndex         int
	IsZeroIndexed      bool
	HasExplicitPageOne bool
	Type               string
}

// PaginationElement represents a clickable pagination element
type PaginationElement struct {
	AriaLabel  string
	Href       string
	Text       string
	PageNumber int    // extracted from aria-label or text
	Confidence string // "high", "medium", "low"
}

func discoverPaginationPattern(page playwright.Page, baseURL *url.URL) (*PaginationPattern, error) {
	logger.Info().Msg("Starting pagination pattern discovery")

	// Step 1: Find all pagination containers
	paginationSelectors := []string{
		".pagination",
		".pager",
		".page-navigation",
		"nav[role='navigation']",
		"[class*='pagination']",
		"[class*='pager']",
		"ul[class*='page']",
	}

	var elements []PaginationElement

	for _, selector := range paginationSelectors {
		containers := page.Locator(selector)
		count, err := containers.Count()
		if err != nil || count == 0 {
			continue
		}

		logger.Info().Str("selector", selector).Int("count", count).Msg("Found pagination containers")

		// Step 2: Extract all anchor tags and buttons within pagination containers
		for i := 0; i < count; i++ {
			container := containers.Nth(i)

			// Find all links and buttons inside this container
			clickables := container.Locator("a, button")
			clickableCount, _ := clickables.Count()

			for j := 0; j < clickableCount; j++ {
				elem := clickables.Nth(j)

				ariaLabel, _ := elem.GetAttribute("aria-label")

				// Check both href and data-href attributes
				// Check both href and data-href and data-page attributes
				href, _ := elem.GetAttribute("href")
				if href == "" {
					href, _ = elem.GetAttribute("data-href")
				}
				if href == "" {
					href, _ = elem.GetAttribute("data-page")
				}
				if href == "" {
					href, _ = elem.GetAttribute("data-url")
				}

				text, _ := elem.TextContent()

				ariaLabel = strings.TrimSpace(ariaLabel)
				href = strings.TrimSpace(href)
				text = strings.TrimSpace(text)

				// Skip if no href or data-href
				if href == "" {
					continue
				}

				// Skip if it's the current/active page
				class, _ := elem.GetAttribute("class")
				if strings.Contains(strings.ToLower(class), "active") ||
					strings.Contains(strings.ToLower(class), "current") ||
					strings.Contains(strings.ToLower(class), "disabled") {
					continue
				}

				// Skip ellipsis
				if text == "..." || text == "…" {
					continue
				}

				pageNum := extractPageNumber(ariaLabel, text)
				if pageNum == -1 && !isNextOrPrevious(ariaLabel, text) {
					continue // Not a pagination element
				}

				confidence := determineConfidence(ariaLabel, href, text, pageNum)

				elements = append(elements, PaginationElement{
					AriaLabel:  ariaLabel,
					Href:       href,
					Text:       text,
					PageNumber: pageNum,
					Confidence: confidence,
				})

				logger.Debug().
					Str("aria-label", ariaLabel).
					Str("href", href).
					Str("text", text).
					Int("page", pageNum).
					Str("confidence", confidence).
					Msg("Found pagination element")
			}
		}
	}

	if len(elements) == 0 {
		return nil, fmt.Errorf("no pagination elements found")
	}

	// Step 3: Analyze collected elements to determine pattern
	// NEW: Pass the page object so we can click elements if needed
	pattern, err := analyzeElementsWithClick(page, elements, baseURL)
	if err != nil {
		return nil, err
	}

	logger.Info().
		Str("param_name", pattern.ParamName).
		Int("start_index", pattern.StartIndex).
		Bool("zero_indexed", pattern.IsZeroIndexed).
		Str("type", pattern.Type).
		Msg("Pagination pattern discovered")

	return pattern, nil
}

// analyzeElementsWithClick analyzes pagination elements and clicks if needed to discover actual URLs
func analyzeElementsWithClick(page playwright.Page, elements []PaginationElement, baseURL *url.URL) (*PaginationPattern, error) {
	logger.Info().Str("function", "analyzeElementsWithClick").Msg("Analyzing pagination elements")

	// Filter for high and medium confidence elements
	var reliable []PaginationElement
	for _, elem := range elements {
		if elem.Confidence == "high" || elem.Confidence == "medium" {
			reliable = append(reliable, elem)
		}
	}

	if len(reliable) == 0 {
		return nil, fmt.Errorf("no reliable pagination elements found")
	}

	// Find page 2 element (most reliable for pattern detection)
	var page2 *PaginationElement
	for i, elem := range reliable {
		if elem.PageNumber == 2 {
			page2 = &reliable[i]
			break
		}
	}

	// If no page 2, try to find "next" button
	if page2 == nil {
		for i, elem := range reliable {
			if isNextOrPrevious(elem.AriaLabel, elem.Text) && elem.Href != "" {
				page2 = &reliable[i]
				page2.PageNumber = 2 // Assume next goes to page 2
				break
			}
		}
	}

	if page2 == nil {
		return nil, fmt.Errorf("could not find page 2 or next button")
	}

	logger.Info().
		Str("href", page2.Href).
		Str("aria-label", page2.AriaLabel).
		Msg("Found page 2 element for pattern analysis")

	// Check if href is just a number (data-href case) or looks incomplete
	isJustNumber := regexp.MustCompile(`^\d+$`).MatchString(page2.Href)

	parsedURL, err := url.Parse(page2.Href)
	isRelativeOrIncomplete := err != nil || !parsedURL.IsAbs()

	// If href is just a number or seems JS-based, we need to click it
	if isJustNumber || isRelativeOrIncomplete {
		logger.Info().
			Str("href", page2.Href).
			Bool("is_just_number", isJustNumber).
			Msg("Href appears to be data-driven or JS-based, will click to discover actual URL")

		// Store current URL before clicking
		currentURL := page.URL()
		logger.Info().Str("current_url", currentURL).Msg("Current URL before clicking")

		// Find the clickable element on the page using aria-label
		var clickableSelector string
		if page2.AriaLabel != "" {
			clickableSelector = fmt.Sprintf("a[aria-label='%s'], button[aria-label='%s']", page2.AriaLabel, page2.AriaLabel)
		} else if page2.Text != "" {
			clickableSelector = fmt.Sprintf("a:has-text('%s'), button:has-text('%s')", page2.Text, page2.Text)
		}

		if clickableSelector == "" {
			return nil, fmt.Errorf("cannot construct selector to click page 2 element")
		}

		clickableElem := page.Locator(clickableSelector).First()

		// Check if element exists
		count, err := clickableElem.Count()
		if err != nil || count == 0 {
			logger.Warn().Str("selector", clickableSelector).Msg("Could not find clickable element")
			return nil, fmt.Errorf("could not find clickable element with selector: %s", clickableSelector)
		}

		logger.Info().Str("selector", clickableSelector).Msg("Clicking pagination element")

		// Click and wait for navigation
		err = clickableElem.Click()
		if err != nil {
			logger.Error().Err(err).Msg("Failed to click pagination element")
			return nil, fmt.Errorf("failed to click pagination element: %w", err)
		}

		// Wait for navigation to complete
		err = page.WaitForLoadState(playwright.PageWaitForLoadStateOptions{
			State: playwright.LoadStateNetworkidle,
		})
		if err != nil {
			logger.Warn().Err(err).Msg("Navigation wait timed out, proceeding anyway")
		}

		// Small additional wait to ensure URL has updated
		time.Sleep(500 * time.Millisecond)

		// Get the new URL after clicking
		newURL := page.URL()
		logger.Info().Str("new_url", newURL).Msg("URL after clicking page 2")

		// Navigate back to page 1
		_, err = page.Goto(currentURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		})
		if err != nil {
			logger.Warn().Err(err).Msg("Failed to navigate back to page 1")
		}

		// Now analyze the actual navigated URL
		return extractPatternFromURL(currentURL, newURL, page2.PageNumber, baseURL)
	}

	// If href looks like a proper URL, use the original analysis method
	return analyzeElements(elements, baseURL)
}

// extractPatternFromURL extracts pagination pattern by comparing two URLs
func extractPatternFromURL(page1URL, page2URL string, page2Number int, baseURL *url.URL) (*PaginationPattern, error) {
	logger.Info().
		Str("page1_url", page1URL).
		Str("page2_url", page2URL).
		Int("page2_number", page2Number).
		Msg("Extracting pattern from actual URLs")

	parsed1, err := url.Parse(page1URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page 1 URL: %w", err)
	}

	parsed2, err := url.Parse(page2URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse page 2 URL: %w", err)
	}

	// Check if URLs are the same (navigation might have failed)
	if page1URL == page2URL {
		return nil, fmt.Errorf("URLs are identical, navigation likely failed")
	}

	pattern := &PaginationPattern{
		BaseURL: baseURL.String(),
	}

	// Compare query parameters
	query1 := parsed1.Query()
	query2 := parsed2.Query()

	// Find which parameter changed
	for param, values2 := range query2 {
		values1 := query1[param]

		// If parameter exists in page 2 but not page 1, or values differ
		if len(values1) == 0 || (len(values2) > 0 && values1[0] != values2[0]) {
			// This is likely the pagination parameter
			if len(values2) > 0 {
				pageValue := values2[0]
				numValue, err := strconv.Atoi(pageValue)

				if err == nil {
					pattern.ParamName = param
					pattern.Type = "query"

					// Determine if zero-indexed
					if numValue == 1 && page2Number == 2 {
						pattern.IsZeroIndexed = true
						pattern.StartIndex = 0
					} else if numValue == 2 && page2Number == 2 {
						pattern.IsZeroIndexed = false
						pattern.StartIndex = 2
					} else {
						// Use the actual value found
						pattern.StartIndex = numValue
						pattern.IsZeroIndexed = (page2Number - numValue) == 1
					}

					logger.Info().
						Str("param", param).
						Str("value", pageValue).
						Bool("zero_indexed", pattern.IsZeroIndexed).
						Int("start_index", pattern.StartIndex).
						Msg("Discovered query parameter pattern from navigation")

					return pattern, nil
				}
			}
		}
	}

	// Check if path changed (path-based pagination)
	if parsed1.Path != parsed2.Path {
		pathPattern := regexp.MustCompile(`/(page/)?(\d+)/?$`)
		matches1 := pathPattern.FindStringSubmatch(parsed1.Path)
		matches2 := pathPattern.FindStringSubmatch(parsed2.Path)

		if len(matches2) > 2 {
			pattern.Type = "path"
			pattern.ParamName = "path"

			pageNum, _ := strconv.Atoi(matches2[2])
			pattern.StartIndex = pageNum
			pattern.IsZeroIndexed = pageNum == 1 && page2Number == 2

			// Check if page 1 also has explicit page number
			if len(matches1) > 2 {
				page1Num, _ := strconv.Atoi(matches1[2])
				pattern.HasExplicitPageOne = page1Num == 1
			}

			logger.Info().
				Str("path", parsed2.Path).
				Int("page_num", pageNum).
				Bool("zero_indexed", pattern.IsZeroIndexed).
				Msg("Discovered path-based pattern from navigation")

			return pattern, nil
		}
	}

	return nil, fmt.Errorf("could not determine pagination pattern from URLs")
}

// analyzeElements - keep the original for cases where href is a proper URL
func analyzeElements(elements []PaginationElement, baseURL *url.URL) (*PaginationPattern, error) {
	logger.Info().Str("function", "analyzeElements").Msg("Analyzing pagination elements")

	// Filter for high and medium confidence elements
	var reliable []PaginationElement
	for _, elem := range elements {
		if elem.Confidence == "high" || elem.Confidence == "medium" {
			reliable = append(reliable, elem)
		}
	}

	if len(reliable) == 0 {
		return nil, fmt.Errorf("no reliable pagination elements found")
	}

	// Find page 2 element (most reliable for pattern detection)
	var page2 *PaginationElement
	for i, elem := range reliable {
		if elem.PageNumber == 2 {
			page2 = &reliable[i]
			break
		}
	}

	// If no page 2, try to find "next" button
	if page2 == nil {
		for i, elem := range reliable {
			if isNextOrPrevious(elem.AriaLabel, elem.Text) && elem.Href != "" {
				page2 = &reliable[i]
				page2.PageNumber = 2 // Assume next goes to page 2
				break
			}
		}
	}

	if page2 == nil {
		return nil, fmt.Errorf("could not find page 2 or next button")
	}

	logger.Info().
		Str("href", page2.Href).
		Str("aria-label", page2.AriaLabel).
		Msg("Using element for pattern analysis")

	// Parse the href to extract pagination pattern
	pattern := &PaginationPattern{
		BaseURL: baseURL.String(),
	}

	if page2.Href == "" {
		return nil, fmt.Errorf("page 2 element has no href")
	}

	// Determine pagination type and extract pattern
	parsedURL, err := url.Parse(page2.Href)
	if err != nil {
		return nil, fmt.Errorf("failed to parse href: %w", err)
	}

	// Make href absolute if it's relative
	if !parsedURL.IsAbs() {
		parsedURL = baseURL.ResolveReference(parsedURL)
	}

	// Check query parameters for pagination
	queryParams := parsedURL.Query()
	paramFound := false

	// Common pagination parameter patterns
	possibleParams := []string{"page", "p", "spage", "paged", "_page", "pageNum", "pageNumber"}

	for _, param := range possibleParams {
		if value := queryParams.Get(param); value != "" {
			pattern.ParamName = param
			pattern.Type = "query"
			paramFound = true

			// Try to determine if it's zero-indexed
			numValue, err := strconv.Atoi(value)
			if err == nil {
				if numValue == 1 && page2.PageNumber == 2 {
					pattern.IsZeroIndexed = true
					pattern.StartIndex = 0
				} else if numValue == 2 && page2.PageNumber == 2 {
					pattern.IsZeroIndexed = false
					pattern.StartIndex = 2
				} else {
					// Check offset
					offset := page2.PageNumber - numValue
					pattern.IsZeroIndexed = offset == 1
					pattern.StartIndex = numValue
				}
			}

			logger.Info().
				Str("param", param).
				Str("value", value).
				Bool("zero_indexed", pattern.IsZeroIndexed).
				Msg("Found query parameter pattern")
			break
		}
	}

	if !paramFound {
		// Check for path-based pagination (/page/2 or /2)
		pathPattern := regexp.MustCompile(`/(page/)?(\d+)/?$`)
		if matches := pathPattern.FindStringSubmatch(parsedURL.Path); len(matches) > 0 {
			pattern.Type = "path"
			pattern.ParamName = "path" // Special marker

			pageNum, _ := strconv.Atoi(matches[2])
			pattern.StartIndex = pageNum
			pattern.IsZeroIndexed = pageNum == 1 && page2.PageNumber == 2

			logger.Info().
				Str("path", parsedURL.Path).
				Int("page_num", pageNum).
				Msg("Found path-based pagination")
			paramFound = true
		}
	}

	if !paramFound {
		return nil, fmt.Errorf("could not determine pagination pattern from href: %s", page2.Href)
	}

	// Validate pattern with additional elements if available
	if len(reliable) > 1 {
		validatePattern(pattern, reliable)
	}

	return pattern, nil
}

// extractPageNumber extracts page number from aria-label or text
func extractPageNumber(ariaLabel, text string) int {
	// Try aria-label first (more reliable)
	if ariaLabel != "" {
		num := extractNumberFromString(ariaLabel)
		if num != -1 {
			return num
		}
	}

	// Fallback to visible text
	if text != "" {
		num := extractNumberFromString(text)
		if num != -1 {
			return num
		}
	}

	return -1
}

// extractNumberFromString extracts a number from a string
func extractNumberFromString(s string) int {
	re := regexp.MustCompile(`\d+`)
	match := re.FindString(s)
	if match != "" {
		num, err := strconv.Atoi(match)
		if err == nil {
			return num
		}
	}
	return -1
}

// isNextOrPrevious checks if element is next/previous button
func isNextOrPrevious(ariaLabel, text string) bool {
	combined := strings.ToLower(ariaLabel + " " + text)
	return strings.Contains(combined, "next") ||
		strings.Contains(combined, "previous") ||
		strings.Contains(combined, "prev") ||
		text == "→" || text == "←" || text == "»" || text == "«"
}

// determineConfidence calculates confidence level for pagination element
func determineConfidence(ariaLabel, href, text string, pageNum int) string {
	hasAriaLabel := ariaLabel != ""
	hasHref := href != "" && href != "#"

	ariaLabelHasPage := strings.Contains(strings.ToLower(ariaLabel), "page")

	// High confidence: aria-label + href + consistent page number
	if hasAriaLabel && hasHref && ariaLabelHasPage && pageNum != -1 {
		// Verify href contains the page number
		if strings.Contains(href, fmt.Sprintf("=%d", pageNum)) ||
			strings.Contains(href, fmt.Sprintf("/%d", pageNum)) {
			return "high"
		}
	}

	// Medium confidence: aria-label with page info + href (even if page number not in href)
	if hasAriaLabel && hasHref && ariaLabelHasPage {
		return "medium"
	}

	// Medium confidence: aria-label with page number, no href (will need to click)
	if hasAriaLabel && ariaLabelHasPage && pageNum != -1 {
		return "medium"
	}

	return "low"
}

// validatePattern validates the discovered pattern against other elements
func validatePattern(pattern *PaginationPattern, elements []PaginationElement) {
	logger.Info().Str("function", "validatePattern").Msg("Validating pagination pattern")

	matches := 0
	for _, elem := range elements {
		if elem.PageNumber == -1 || elem.Href == "" {
			continue
		}

		parsedURL, err := url.Parse(elem.Href)
		if err != nil {
			continue
		}

		if pattern.Type == "query" {
			value := parsedURL.Query().Get(pattern.ParamName)
			numValue, err := strconv.Atoi(value)
			if err == nil {
				expectedValue := elem.PageNumber
				if pattern.IsZeroIndexed {
					expectedValue = elem.PageNumber - 1
				}

				if numValue == expectedValue {
					matches++
				}
			}
		}
	}

	logger.Info().Int("matches", matches).Int("total", len(elements)).Msg("Pattern validation")
}

// generatePaginatedURL generates a URL for a specific page number
func generatePaginatedURL(pattern *PaginationPattern, pageNumber int) (string, error) {
	if pattern == nil {
		return "", fmt.Errorf("pagination pattern is nil")
	}

	baseURL, err := url.Parse(pattern.BaseURL)
	if err != nil {
		return "", fmt.Errorf("failed to parse base URL: %w", err)
	}

	switch pattern.Type {
	case "query":
		query := baseURL.Query()

		paramValue := pageNumber
		if pattern.IsZeroIndexed {
			paramValue = pageNumber - 1
		}

		query.Set(pattern.ParamName, strconv.Itoa(paramValue))
		baseURL.RawQuery = query.Encode()

		return baseURL.String(), nil

	case "path":
		// Handle path-based pagination
		path := baseURL.Path

		// Remove trailing slash
		path = strings.TrimSuffix(path, "/")

		paramValue := pageNumber
		if pattern.IsZeroIndexed {
			paramValue = pageNumber - 1
		}

		// Check if path already has /page/ pattern
		if strings.Contains(path, "/page/") {
			re := regexp.MustCompile(`/page/\d+`)
			path = re.ReplaceAllString(path, fmt.Sprintf("/page/%d", paramValue))
		} else {
			path = fmt.Sprintf("%s/page/%d", path, paramValue)
		}

		baseURL.Path = path
		return baseURL.String(), nil

	default:
		return "", fmt.Errorf("unsupported pagination type: %s", pattern.Type)
	}
}

// scanForJobsWithPagination scans for jobs with automatic pagination support
func scanForJobsWithPagination(page playwright.Page, baseURL *url.URL, maxPages int) []models.LinkData {
	logger.Info().Msg("Starting job scan with pagination")

	var allJobs []models.LinkData
	seenURLs := make(map[string]bool)

	// Scan first page
	jobs := scanForJobs(page, baseURL)
	for _, job := range jobs {
		if !seenURLs[job.URL] {
			seenURLs[job.URL] = true
			allJobs = append(allJobs, job)
		}
	}

	logger.Info().Int("jobs_page_1", len(jobs)).Msg("Scanned first page")

	// Discover pagination pattern
	pattern, err := discoverPaginationPattern(page, baseURL)
	if err != nil {
		logger.Warn().Err(err).Msg("No pagination found, returning first page results")
		return allJobs
	}

	// Paginate through remaining pages
	for pageNum := 2; pageNum <= maxPages; pageNum++ {
		nextURL, err := generatePaginatedURL(pattern, pageNum)
		if err != nil {
			logger.Error().Err(err).Int("page", pageNum).Msg("Failed to generate URL")
			break
		}

		logger.Info().Int("page", pageNum).Str("url", nextURL).Msg("Navigating to next page")

		_, err = page.Goto(nextURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		})
		if err != nil {
			logger.Error().Err(err).Int("page", pageNum).Msg("Failed to navigate")
			break
		}

		// Scan for jobs on this page
		jobs = scanForJobs(page, baseURL)

		if len(jobs) == 0 {
			logger.Info().Int("page", pageNum).Msg("No jobs found, reached end")
			break
		}

		// Add unique jobs
		newJobs := 0
		for _, job := range jobs {
			if !seenURLs[job.URL] {
				seenURLs[job.URL] = true
				allJobs = append(allJobs, job)
				newJobs++
			}
		}

		logger.Info().Int("page", pageNum).Int("new_jobs", newJobs).Int("total", len(allJobs)).Msg("Scanned page")

		// If no new jobs found, we might be seeing duplicates (end of pagination)
		if newJobs == 0 {
			logger.Info().Int("page", pageNum).Msg("No new jobs, reached end")
			break
		}
	}

	logger.Info().Int("total_jobs", len(allJobs)).Msg("Completed pagination scan")
	return allJobs
}

/* ================= HASH / SPA WAIT ================= */

func waitForJobContent(page playwright.Page) {
	page.WaitForFunction(`
	() => {
		const t = document.body.innerText.toLowerCase();
		return (
			t.includes("engineer") ||
			t.includes("developer") ||
			t.includes("manager") ||
			t.includes("senior")
		);
	}
	`, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(float64(jobWaitTimeout.Milliseconds())),
	})
}

/* ================= HELPERS ================= */

func containsAny(text string, words []string) bool {
	for _, w := range words {
		if strings.Contains(text, w) {
			return true
		}
	}
	return false
}

func dedupeJobs(jobs []models.LinkData) []models.LinkData {
	seen := make(map[string]bool)
	var out []models.LinkData

	for _, job := range jobs {
		if job.URL == "" || seen[job.URL] {
			continue
		}
		seen[job.URL] = true
		out = append(out, job)
	}
	return out
}

func logJobs(jobs []models.LinkData) {
	logger.Info().Int("Job listings:", len(jobs))
	for i, job := range jobs {
		logger.Info().Int("number", i+1).Str("text", job.Text).Str("url", job.URL).Msg("Job listing")
	}
}

func toJSArray(arr []string) string {
	logger.Info().Msg("converting to JS array in toJSArray func")
	q := make([]string, len(arr))
	for i, s := range arr {
		q[i] = `"` + s + `"`
	}
	return "[" + strings.Join(q, ",") + "]"
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func resolveURL(href string, base *url.URL) string {
	logger.Info().Str("href", href).Msg("resolving URL")
	u, err := url.Parse(href)
	if err != nil {
		return href
	}
	return base.ResolveReference(u).String()
}
