package service

import (
	"fmt"
	"log/slog"
	"net/url"
	"regexp"
	"strings"

	"github.com/playwright-community/playwright-go"
)

var Xpath = `xpath=//`

func ScrapeJobs(browser BrowserService, companyUrl string, scid uint) {
	page, err := browser.RunInNewTab()
	if err != nil {
		slog.Error("Error creating new page", slog.String("of", companyUrl), slog.Any("is", err))
		return
	}
	defer page.Close()

	careersUrl, err := navigateToCareersPage(page, companyUrl)
	if err != nil {
		slog.Error("Error finding careers page", slog.String("of", companyUrl), slog.Any("is", err))
		return
	}

	_, err = page.Goto(careersUrl, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		slog.Error("Error loading careers page", slog.String("url", careersUrl), slog.Any("is", err))
		return
	}

	slog.Info("Ready to scrape jobs", slog.String("url", careersUrl))
	locator := page.Locator(Xpath)

	count, err := locator.Count()
	if err != nil || count == 0 {
		slog.Info("No testimonial images found for", slog.String("company url", companyUrl))
		return
	}
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
			slog.Info("Found careers page via direct URL", slog.String("url", targetUrl))
			return targetUrl, nil
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
		"Careers",
		"careers",
		"Jobs",
		"jobs",
		"Join Us",
		"Join us",
		"Work with us",
		"Work With Us",
		"We're Hiring",
		"We're hiring",
	}

	for _, text := range careersTexts {
		// Find element containing the text
		locator := page.Locator(fmt.Sprintf(`text="%s"`, text)).First()

		count, err := locator.Count()
		if err != nil || count == 0 {
			continue
		}

		href, err := locator.GetAttribute("href")
		if err == nil && href != "" {
			resolved := resolveUrl(companyUrl, href)
			if resolved != "" {
				slog.Info("Found careers link on element", slog.String("href", resolved))
				return resolved, nil
			}
		}

		ancestorAnchor := locator.Locator("xpath=ancestor::a").First()
		count, err = ancestorAnchor.Count()
		if err == nil && count > 0 {
			href, err = ancestorAnchor.GetAttribute("href")
			if err == nil && href != "" {
				resolved := resolveUrl(companyUrl, href)
				if resolved != "" {
					slog.Info("Found careers link via ancestor anchor", slog.String("href", resolved))
					return resolved, nil
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
					slog.Info("Found careers link via child anchor", slog.String("href", resolved))
					return resolved, nil
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
				slog.Info("Found careers link via href selector", slog.String("href", resolved))
				return resolved, nil
			}
		}
	}

	return "", fmt.Errorf("no careers page found for %s", companyUrl)
}

func resolveUrl(baseUrl, href string) string {
	if strings.HasPrefix(href, "javascript:") {
		url := extractUrlFromJavascript(href)
		if url != "" {
			return resolveUrl(baseUrl, url)
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
	if strings.HasPrefix(href, "/") {
		return baseUrl + href
	}
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
