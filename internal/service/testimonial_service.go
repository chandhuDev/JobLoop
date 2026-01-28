package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/url"
	"strings"
	"sync"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
)

type TestimonialService struct {
	Testimonial *models.Testimonial
}

func NewTestimonial() *models.Testimonial {
	return &models.Testimonial{
		ImageResultChan: make(chan models.TestimonialImageResult, 250),
		TestimonialWg:   &sync.WaitGroup{},
		ImageWg:         &sync.WaitGroup{},
	}
}

func (t *TestimonialService) ScrapeTestimonial(
	ctx context.Context,
	scraper *interfaces.ScraperClient,
	dupChan <-chan models.SeedCompanyResult,
	// scChan <-chan models.SeedCompanyResult,
	vision VisionWrapper,
) {
	numTestimonialWorkers := 2
	numImageWorkers := 2

	// Testimonial scraper workers
	for i := 0; i < numTestimonialWorkers; i++ {
		t.Testimonial.TestimonialWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.TestimonialWg.Done()
			slog.Info("Starting Testimonial worker", slog.Int("worker_id", workerID))

			page, err := scraper.Browser.RunInNewTab()
			if err != nil {
				slog.Error("Failed to create page", slog.Int("worker_id", workerID), slog.Any("error", err))
				return
			}
			defer func() {
				slog.Info("Closing page", slog.Int("worker_id", workerID))
				page.Close()
			}()

			for {
				select {
				case <-ctx.Done():
					slog.Info("Testimonial worker stopping (context cancelled)", slog.Int("worker_id", workerID))
					return
				case scr, ok := <-dupChan:
					if !ok {
						slog.Info("Testimonial worker stopping (channel closed)", slog.Int("worker_id", workerID))
						return
					}

					slog.Info("Processing", slog.Int("worker", workerID), slog.String("company", scr.CompanyName))

					urls := t.scrapeCompany(ctx, page, scr)
					if len(urls) > 0 {
						select {
						case t.Testimonial.ImageResultChan <- models.TestimonialImageResult{
							SeedCompanyId: scr.SeedCompanyId,
							CompanyName:   scr.CompanyName,
							URL:           urls,
						}:
						case <-ctx.Done():
							slog.Info("Testimonial worker stopping during send", slog.Int("worker_id", workerID))
							return
						}
					}
				}
			}
		}(i)
	}

	// Image processing workers
	for i := 0; i < numImageWorkers; i++ {
		t.Testimonial.ImageWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.ImageWg.Done()
			slog.Info("Starting Image worker", slog.Int("worker_id", workerID))

			for {
				select {
				case <-ctx.Done():
					slog.Info("Image worker stopping (context cancelled)", slog.Int("worker_id", workerID))
					return
				case job, ok := <-t.Testimonial.ImageResultChan:
					if !ok {
						slog.Info("Image worker stopping (channel closed)", slog.Int("worker_id", workerID))
						return
					}

					slog.Info("Processing images",
						slog.Int("worker", workerID),
						slog.String("company", job.CompanyName),
						slog.Int("count", len(job.URL)))

					for _, url := range job.URL {
						slog.Info("Extracting text from image", slog.String("url", url))
					}

					//  vision.ExtractTextFromImage(job.URL, scraper, workerID, job.SeedCompanyId)
				}
			}
		}(i)
	}

	// Wait for testimonial workers, then close image channel
	t.Testimonial.TestimonialWg.Wait()
	slog.Info("All testimonial workers finished, closing image channel")
	close(t.Testimonial.ImageResultChan)

	// Wait for image workers
	t.Testimonial.ImageWg.Wait()
	slog.Info("All image workers finished")
}

func (t *TestimonialService) scrapeCompany(
	ctx context.Context,
	page playwright.Page,
	scr models.SeedCompanyResult,
) []string {

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	pageURL := scr.CompanyURL
	if !strings.HasPrefix(pageURL, "http://") && !strings.HasPrefix(pageURL, "https://") {
		pageURL = "https://" + pageURL
	}

	slog.Info("Navigating to", slog.String("url", pageURL))

	// Use 'load' instead of 'networkidle' - networkidle can timeout on heavy sites
	resp, err := page.Goto(pageURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(30000), // Reduce timeout
	})
	
	// Check if we got blocked (403) vs just timeout
	if resp != nil {
		status := resp.Status()
		if status == 403 || status == 401 {
			slog.Warn("Access denied", slog.Int("status", status), slog.String("company", scr.CompanyName))
			return nil
		}
		slog.Info("Page response", slog.Int("status", status))
	}
	
	// Timeout is OK - page might still have loaded
	if err != nil {
		slog.Warn("Navigation warning (continuing anyway)", slog.String("url", pageURL), slog.Any("error", err))
	}

	// Wait for body to exist
	_, err = page.WaitForFunction(`
		() => document.body && document.body.children.length > 0
	`, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(10000),
	})
	if err != nil {
		slog.Error("DOM never became ready", slog.Any("error", err))
		return nil
	}

	// Wait for images to load
	page.WaitForTimeout(3000)

	// Scroll to trigger lazy loading
	page.Evaluate(`
		() => {
			return new Promise((resolve) => {
				let scrollCount = 0;
				const maxScrolls = 5;
				const timer = setInterval(() => {
					window.scrollBy(0, 500);
					scrollCount++;
					if (scrollCount >= maxScrolls) {
						clearInterval(timer);
						window.scrollTo(0, 0);
						setTimeout(() => resolve(true), 500);
					}
				}, 200);
			});
		}
	`)

	// Additional wait after scrolling
	page.WaitForTimeout(2000)

	select {
	case <-ctx.Done():
		return nil
	default:
	}

	// Check if page is actually loaded (not blocked)
	title, _ := page.Title()
	if strings.Contains(strings.ToLower(title), "access denied") || 
	   strings.Contains(strings.ToLower(title), "blocked") ||
	   strings.Contains(strings.ToLower(title), "forbidden") {
		slog.Warn("Page blocked", slog.String("title", title), slog.String("company", scr.CompanyName))
		return nil
	}

	count, _ := page.Evaluate(`() => document.querySelectorAll("img").length`)
	slog.Info("IMG COUNT", slog.Any("count", count), slog.String("company", scr.CompanyName))

	jsonStr, err := scrapeTestimonialImageUrls(page)
	if err != nil {
		slog.Error("JS evaluation failed", slog.String("company", scr.CompanyName), slog.Any("error", err))
		return nil
	}

	type testimonialJSResult struct {
		Found  bool     `json:"found"`
		Phase  string   `json:"phase"`
		Count  int      `json:"count"`
		Images []string `json:"images"`
	}

	var data testimonialJSResult

	if err := json.Unmarshal([]byte(jsonStr), &data); err != nil {
		slog.Error("Failed to unmarshal JS result", slog.Any("error", err))
		return nil
	}

	if !data.Found || len(data.Images) == 0 {
		slog.Warn("No testimonial images found", slog.String("company", scr.CompanyName))
		return nil
	}

	var normalized []string
	for _, src := range data.Images {
		full := toAbsoluteURL(pageURL, src)
		if full != "" {
			normalized = append(normalized, full)
		}
	}

	if len(normalized) == 0 {
		slog.Warn("Images extracted but empty after normalization", slog.String("company", scr.CompanyName))
		return nil
	}

	slog.Info(
		"Testimonial images found",
		slog.String("company", scr.CompanyName),
		slog.String("phase", data.Phase),
		slog.Int("count", len(normalized)),
	)

	return normalized
}

func toAbsoluteURL(baseURL, src string) string {
	if src == "" {
		return ""
	}

	if strings.HasPrefix(src, "http://") || strings.HasPrefix(src, "https://") {
		return src
	}

	if strings.HasPrefix(src, "data:") {
		return ""
	}

	parsed, err := url.Parse(baseURL)
	if err != nil {
		return src
	}

	if strings.HasPrefix(src, "//") {
		return parsed.Scheme + ":" + src
	}

	if strings.HasPrefix(src, "/") {
		return parsed.Scheme + "://" + parsed.Host + src
	}

	return parsed.Scheme + "://" + parsed.Host + "/" + src
}

func scrapeTestimonialImageUrls(page playwright.Page) (string, error) {
	count, _ := page.Evaluate(`() => document.querySelectorAll("img").length`)
	slog.Warn("IMG COUNT BEFORE EVAL", slog.Any("count", count))
	result, err := page.Evaluate(`
	() => {
	  /* ================= CONFIG ================= */

	  const TRUST_KEYWORDS = [
		'trusted by',
		'powered by',
		'our customers',
		'integrated by',
		'used by',
		'loved by',
		'works with',
		'supported platforms'
	  ];

	  const SLIDER_HINTS = [
		'slider',
		'carousel',
		'swiper',
		'slick',
		'marquee',
		'infinite'
	  ];

	  const BAD_URL_HINTS = [
		'sanity.io',
		'wistia',
		'dashboard',
		'screenshot',
		'thumbnail',
		'hero',
		'banner',
		'background',
		'mockup',
		'platform',
		'feature',
		'overview',
		'blur(',
		'quality(0)',
		'data:image'
	  ];

	  const ICON_HINTS = [
		'icon',
		'integrate',
		'verify',
		'locate',
		'enrich',
		'engage',
		'analyze',
		'connect',
		'build',
		'manage',
		'secure'
	  ];

	  /* ================= HELPERS ================= */

	  const safeLower = v => String(v || '').toLowerCase();

	  const isVisible = el => {
		const s = window.getComputedStyle(el);
		return s && s.display !== 'none' && s.visibility !== 'hidden';
	  };

	  const isInHeaderFooterNav = el => {
		let cur = el;
		while (cur && cur !== document.body) {
		  const tag = cur.tagName?.toLowerCase();
		  if (['header', 'footer', 'nav'].includes(tag)) return true;

		  const cls = safeLower(cur.className);
		  const id  = safeLower(cur.id);

		  if (
			cls.includes('header') ||
			cls.includes('footer') ||
			cls.includes('nav') ||
			id.includes('header') ||
			id.includes('footer') ||
			id.includes('nav')
		  ) return true;

		  cur = cur.parentElement;
		}
		return false;
	  };

	  const looksLikeLogoSize = el => {
		const r = el.getBoundingClientRect();
		if (!r || r.width === 0 || r.height === 0) return false;
		if (r.width < 48 || r.height < 20) return false;

		const ratio = r.width / r.height;
		if (ratio < 0.25 || ratio > 6) return false;

		if (r.width > 700 && r.height > 500) return false;
		return true;
	  };

	  const unwrapNextImage = src => {
		if (!src.includes('/_next/image')) return src;
		try {
		  const u = new URL(src, location.origin);
		  const inner = u.searchParams.get('url');
		  return inner ? decodeURIComponent(inner) : src;
		} catch {
		  return src;
		}
	  };

	  const isNoiseUrl = src =>
		BAD_URL_HINTS.some(h => safeLower(src).includes(h));

	  const isFeatureIconUrl = src =>
		ICON_HINTS.some(h => safeLower(src).includes(h));

	  const extractVisuals = root => {
		const results = [];

		root.querySelectorAll('img').forEach(img => {
		  if (isInHeaderFooterNav(img)) return;

		  let src =
  img.src ||
  img.currentSrc ||
  img.dataset?.src ||
  img.getAttribute('data-src') ||
  img.getAttribute('data-lazy-src') ||
  '';


		  if (!src) return;
		  src = unwrapNextImage(src);

		  if (isNoiseUrl(src)) return;
		  if (isFeatureIconUrl(src)) return;
		  if (!looksLikeLogoSize(img)) return;

		  results.push(src);
		});

		root.querySelectorAll('svg').forEach(svg => {
		  if (isInHeaderFooterNav(svg)) return;
		  if (!looksLikeLogoSize(svg)) return;

		  const use = svg.querySelector('use');
		  if (!use) return;

		  let href =
			use.getAttribute('href') ||
			use.getAttribute('xlink:href') ||
			'';

		  if (!href) return;

		  if (href.startsWith('/')) href = location.origin + href;
		  if (isNoiseUrl(href)) return;
		  if (isFeatureIconUrl(href)) return;

		  results.push(href);
		});

		return results;
	  };

	  const dedupe = arr => [...new Set(arr)];

	  /* ================= PHASE 1 ================= */

	  const trustTextNodes = [];
	  const walker = document.createTreeWalker(
		document.body,
		NodeFilter.SHOW_TEXT,
		{
		  acceptNode(node) {
			const t = safeLower(node.textContent).trim();
			if (!t) return NodeFilter.FILTER_REJECT;
			if (!node.parentElement || !isVisible(node.parentElement))
			  return NodeFilter.FILTER_REJECT;

			return TRUST_KEYWORDS.some(k => t.includes(k))
			  ? NodeFilter.FILTER_ACCEPT
			  : NodeFilter.FILTER_REJECT;
		  }
		}
	  );

	  while (walker.nextNode()) trustTextNodes.push(walker.currentNode);

	  const semanticImages = [];

	  trustTextNodes.forEach(node => {
		const visited = new Set();
		const queue = [node.parentElement];

		while (queue.length) {
		  const el = queue.shift();
		  if (!el || visited.has(el)) continue;
		  visited.add(el);

		  extractVisuals(el).forEach(v => semanticImages.push(v));

		  Array.from(el.children || []).forEach(c => queue.push(c));
		  if (el.parentElement) queue.push(el.parentElement);
		  if (el.nextElementSibling) queue.push(el.nextElementSibling);
		  if (el.previousElementSibling) queue.push(el.previousElementSibling);
		}
	  });

	  const uniqueSemantic = dedupe(semanticImages);

	  if (uniqueSemantic.length > 0) {
		return JSON.stringify({
		  found: true,
		  phase: 'semantic',
		  count: uniqueSemantic.length,
		  images: uniqueSemantic
		});
	  }

	  /* ================= PHASE 2 ================= */

	  const sliderContainers = Array.from(document.querySelectorAll('*')).filter(el => {
		const cls = safeLower(el.className);
		const style = window.getComputedStyle(el);

		if (SLIDER_HINTS.some(h => cls.includes(h))) return true;
		if (style.overflowX === 'hidden' && el.children.length >= 3) return true;
		return false;
	  });

	  const sliderImages = [];
	  sliderContainers.forEach(c =>
		extractVisuals(c).forEach(v => sliderImages.push(v))
	  );

	  const uniqueSlider = dedupe(sliderImages);

	  if (uniqueSlider.length > 0) {
		return JSON.stringify({
		  found: true,
		  phase: 'slider',
		  count: uniqueSlider.length,
		  images: uniqueSlider
		});
	  }

	  return JSON.stringify({
		found: false,
		phase: 'none',
		reason: 'no reliable logo images detected'
	  });
	}
`)
	if err != nil {
		return "", err
	}

	jsonStr, ok := result.(string)
	if !ok {
		return "", fmt.Errorf("expected JSON string from evaluate")
	}

	return jsonStr, nil
}
