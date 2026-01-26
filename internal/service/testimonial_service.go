package service

import (
	"log/slog"
	"net/url"
	"strings"
	"sync"
	"time"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/playwright-community/playwright-go"
)

type TestimonialService struct {
	Testimonial *models.Testimonial
}

func NewTestimonial() *models.Testimonial {
	return &models.Testimonial{
		ImageResultChan: make(chan []string, 500),
		TestimonialWg:   &sync.WaitGroup{},
		ImageWg:         &sync.WaitGroup{},
	}
}

func (t *TestimonialService) ScrapeTestimonial(scraper *interfaces.ScraperClient, scChan <-chan models.SeedCompanyResult, vision VisionWrapper) {
	for i := 0; i < 2; i++ {
		t.Testimonial.TestimonialWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.TestimonialWg.Done()
			slog.Info("Starting Testimonial goroutine", slog.Int("goroutine id", workerID))

			page, err := scraper.Browser.RunInNewTab()
			if err != nil {
				scraper.Err.Send(models.WorkerError{
					WorkerId: workerID,
					Message:  "Error creating new page",
					Err:      err,
				})
				return
			}
			defer page.Close()

			for scr := range scChan {
				t.Testimonial.SeedCompanyId = scr.SeedCompanyId
				slog.Info("START processing", slog.String("company", scr.CompanyName), slog.Time("time", time.Now()))
				url := scr.CompanyURL

				if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
					url = "https://" + url
				}

				_, err := page.Goto(url, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateNetworkidle,
					Timeout:   playwright.Float(60000),
				})
				if err != nil {
					slog.Error("Navigation failed", slog.String("company", scr.CompanyName), slog.String("url", scr.CompanyURL), slog.Any("error", err))
					scraper.Err.Send(models.WorkerError{
						WorkerId: workerID,
						Message:  "Error navigating to testimonial page",
						Err:      err,
					})
					continue
				}
				// slog.Info("Navigation successful", slog.String("company", scr.CompanyName))

				result, err := scrapeTestimonialImageUrls(page)
				if err != nil {
					slog.Error("JavaScript evaluation failed", slog.Any("error", err))
					return
				}

				if result == nil {
					slog.Warn("No result from JavaScript evaluation")
					return
				}

				data, ok := result.(map[string]interface{})
				if !ok {
					slog.Warn("Unexpected result format")
					return
				}

				found, _ := data["found"].(bool)
				if !found {
					slog.Warn("No testimonial images found", slog.String("company", scr.CompanyName))
					return
				}
				count, _ := data["count"].(float64)
				level, _ := data["level"].(float64)
				tagName, _ := data["tagName"].(string)
				className, _ := data["className"].(string)

				slog.Info("Best group found",
					slog.Int("count", int(count)),
					slog.Int("level", int(level)),
					slog.String("tagName", tagName),
					slog.String("className", className))

				images, ok := data["images"].([]interface{})
				if !ok || len(images) == 0 {
					slog.Warn("No images in result")
					return
				}

				var urlArray []string
				maxImages := 10 // Limit to 10 images per company to avoid overwhelming Vision API
				for i, img := range images {
					if i >= maxImages {
						break
					}
					if src, ok := img.(string); ok && src != "" {
						fullURL := toAbsoluteURL(url, src)
						if fullURL != "" {
							urlArray = append(urlArray, fullURL)
						}
					}
				}

				if len(urlArray) > 0 {
					slog.Info("Found testimonial images", slog.String("company", scr.CompanyName), slog.Int("count", len(urlArray)))
					t.Testimonial.ImageResultChan <- urlArray
				} else {
					slog.Warn("No valid image URLs found", slog.String("company", scr.CompanyName))
				}

			}
		}(i)
	}
	
	for i := 0; i < 2; i++ {
		t.Testimonial.ImageWg.Add(1)
		go func(workerID int) {
			defer t.Testimonial.ImageWg.Done()

			for urlArray := range t.Testimonial.ImageResultChan {
				slog.Info("Processing images", slog.Int("worker_id", workerID), slog.Int("url_count", len(urlArray)))
				vision.ExtractTextFromImage(urlArray, scraper, workerID, t.Testimonial.SeedCompanyId)
			}
		}(i)
	}

	t.Testimonial.TestimonialWg.Wait()
	close(t.Testimonial.ImageResultChan)
	t.Testimonial.ImageWg.Wait()
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

func scrapeTestimonialImageUrls(page playwright.Page) (interface{}, error) {
	result, err := page.Evaluate(`() => {
		const allImgs = document.querySelectorAll('img');
		
		const excludePatterns = ['header', 'footer', 'nav', 'menu', 'social', 'icon'];
		
		const isExcluded = (el) => {
			let current = el;
			while (current && current !== document.body) {
				const className = (current.className || '').toLowerCase();
				const id = (current.id || '').toLowerCase();
				const tag = current.tagName.toLowerCase();
				
				for (const pattern of excludePatterns) {
					if (className.includes(pattern) || id.includes(pattern) || tag === pattern) {
						return true;
					}
				}
				current = current.parentElement;
			}
			return false;
		};
		
		const imgs = Array.from(allImgs).filter(img => !isExcluded(img));
		
		const findSiblingGroups = (img) => {
			const results = [];
			
			for (let level = 0; level <= 3; level++) {
				let element = img;
				for (let i = 0; i < level; i++) {
					element = element.parentElement;
					if (!element) break;
				}
				if (!element || !element.parentElement) continue;
				
				const parent = element.parentElement;
				const className = element.className || '';
				const tagName = element.tagName;
				
				const siblings = Array.from(parent.children).filter(child => {
					return child.tagName === tagName && child.className === className;
				});
				
				if (siblings.length < 3) continue;
				
				let hasTextBetween = false;
				parent.childNodes.forEach(node => {
					if (node.nodeType === Node.TEXT_NODE && node.textContent.trim().length > 0) {
						hasTextBetween = true;
					}
				});
				
				if (!hasTextBetween) {
					results.push({
						level,
						className,
						tagName,
						siblingCount: siblings.length
					});
				}
			}
			
			return results;
		};
		
		const allGroups = new Map();
		
		imgs.forEach(img => {
			const groups = findSiblingGroups(img);
			groups.forEach(group => {
				const key = group.level + '_' + group.tagName + '_' + group.className;
				if (!allGroups.has(key)) {
					allGroups.set(key, {
						level: group.level,
						tagName: group.tagName,
						className: group.className,
						images: []
					});
				}
				const entry = allGroups.get(key);
				const src = img.src || img.dataset?.src || img.getAttribute('data-lazy-src') || '';
				if (src && !entry.images.includes(src)) {
					entry.images.push(src);
				}
			});
		});
		
		const sorted = Array.from(allGroups.values())
			.filter(g => g.images.length >= 3)
			.sort((a, b) => {
				if (b.images.length !== a.images.length) {
					return b.images.length - a.images.length;
				}
				return a.level - b.level;
			});
		
		if (sorted.length > 0) {
			return {
				found: true,
				count: sorted[0].images.length,
				level: sorted[0].level,
				tagName: sorted[0].tagName,
				className: sorted[0].className,
				images: sorted[0].images
			};
		}
		
		return { found: false, images: [] };
	}`)
	return result, err
}
