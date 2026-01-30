package service

import (
	"fmt"
	"net/url"
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
		"view all", "view jobs", "view openings", "browse jobs", "browse openings", "view roles", "view positions", "open positions", "see openings", "see jobs", "search jobs", "explore open roles", "explore jobs", "join us", "see all jobs", "see our open roles",
	}

	jobKeywords = []string{
		"senior", "lead", "engineer", "experience", "sales",
		"platform", "head", "product", "manager", "principal",
		"strategy", "solutions", "partner", "employee", "finance",
		"account", "executive", "consultant", "sr.", "strategic", "operations", "acquisition", "business", "analyst", "customer",
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

	ctas := findCTAs(page)
	logger.Info().Int("cta_count", len(ctas)).Msg("Found CTA buttons")

	/* ---------- HANDLE EACH CTA ---------- */

	for _, cta := range ctas {
		logger.Info().Str("cta_text", cta.Text).Str("cta_href", cta.RawHref).Msg("CTA")

		if strings.HasPrefix(cta.RawHref, "#") {
			logger.Info().Msg("Hash CTA detected - scraping current page only")

			jobs := scanForJobs(page)
			jobs = dedupeJobs(jobs)

			if len(jobs) > 0 {
				logger.Info().Msg("Jobs found on careers page (hash CTA)")
				logJobs(jobs)
				return jobs, nil
			}

			continue
		}

		target := resolveURL(cta.RawHref, baseURL)
		logger.Info().Str("cta_url", target).Msg("Navigating CTA URL")

		_, err = page.Goto(target, playwright.PageGotoOptions{
			Timeout: playwright.Float(30000),
		})
		if err != nil {
			logger.Warn().Str("url", target).Err(err).Msg("Failed to navigate to CTA")
			continue
		}

		waitForJobContent(page)

		jobs := scanForJobs(page)
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

		jobs := scanForJobs(page)
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

func scanForJobs(page playwright.Page) []models.LinkData {
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

		logger.Debug().Str("href", href).Msg("Processing anchor")

		// 1️⃣ Anchor text
		text, _ := a.TextContent()
		text = strings.TrimSpace(text)

		// 2️⃣ Fallback to container text
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

		if seen[href] {
			continue
		}
		seen[href] = true

		logger.Info().Str("href", href).Str("text", text).Msg("Job found")

		jobs = append(jobs, models.LinkData{
			Text: text,
			URL:  href,
		})
	}

	logger.Info().Int("jobs", len(jobs)).Msg("Jobs found after scan")
	return jobs
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
	logger.Info().Msg("Job listings:")
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

func parseJobs(res interface{}) []models.LinkData {
	logger.Info().Msg("we are in parseJobs func")
	var out []models.LinkData
	arr, ok := res.([]interface{})
	if !ok {
		return out
	}

	for _, v := range arr {
		m := v.(map[string]interface{})
		out = append(out, models.LinkData{
			Text: toString(m["text"]),
			URL:  toString(m["href"]),
		})
		logger.Debug().Str("text", toString(m["text"])).Str("url", toString(m["href"])).Msg("Found job")
	}
	return out
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
