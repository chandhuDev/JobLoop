package service

import (
	"context"
	"log/slog"
	"sync"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

type TestimonialService struct {
	Testimonial *models.Testimonial
}

func NewTestimonial() *models.Testimonial {
	return &models.Testimonial{
		ImageResultChan: make(chan []string, 100),
		TestimonialWg:   sync.WaitGroup{},
		ImageWg:         sync.WaitGroup{},
	}
}

func (t *TestimonialService) ScrapeTestimonial(scraper *interfaces.ScraperClient, scChan <-chan models.SeedCompanyResult, vision VisionWrapper) {

	for i := 0; i < 5; i++ {
		t.Testimonial.TestimonialWg.Add(1)
		go func(i int, browser interfaces.BrowserClient, scChan <-chan models.SeedCompanyResult, wg *sync.WaitGroup, im chan []string, e interfaces.ErrorClient) {

			defer t.Testimonial.TestimonialWg.Done()
			tabContext, tabCancel := scraper.Browser.RunInNewTab()
			defer tabCancel()
			xpath := `
		  (
			//*[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'trust')]
			  /following::*[count(.//img) >= 3][1]
		  )//img
		  `

			for scr := range scChan {
				var nodes []*cdp.Node

				err := chromedp.Run(tabContext,
					chromedp.Navigate(scr.CompanyURL),
					chromedp.WaitVisible("body"),
					chromedp.Nodes(xpath, &nodes, chromedp.BySearch, chromedp.AtLeast(0)),
				)
				if err != nil {
					e.Send(models.WorkerError{
						WorkerId: i,
						Message:  "Error navigating to testimonial page:",
						Err:      err,
					})
				}
				if len(nodes) == 0 || nodes == nil {
					slog.Info("No testimonial images found for", scr.CompanyName)
					break
				}
				var UrlArray []string

				for j := range nodes {
					var fullURL string
					fullURL = getAttr(tabContext, nodes[j].FullXPath(), "src", e)
					if fullURL == "" || fullURL == "null" {
						fullURL = getAttr(tabContext, nodes[j].FullXPath(), "data-src", e)
					}
					UrlArray = append(UrlArray, fullURL)
				}
				im <- UrlArray

			}
		}(i, scraper.Browser, scChan, &t.Testimonial.TestimonialWg, t.Testimonial.ImageResultChan, scraper.Err)

	}

	for i := 0; i < 5; i++ {
		t.Testimonial.ImageWg.Add(1)
		go func(i int, v VisionWrapper, e interfaces.ErrorClient) {
			slog.Info("Starting image processor goroutine %d\n", i)
			defer t.Testimonial.TestimonialWg.Done()
			for urlArray := range t.Testimonial.ImageResultChan {
				v.ExtractImageFromText(urlArray, e)
			}
		}(i, vision, scraper.Err)
	}

	go func() {
		t.Testimonial.TestimonialWg.Wait()
		close(t.Testimonial.ImageResultChan)
	}()
	t.Testimonial.ImageWg.Wait()

}

func getAttr(ctx context.Context, xpath string, attributeName string, e interfaces.ErrorClient) string {
	var url string
	err := chromedp.Run(ctx, chromedp.JavascriptAttribute(xpath, attributeName, &url))
	if err != nil {
		e.Send(models.WorkerError{
			WorkerId: -1,
			Message:  "Error getting JS attribute " + attributeName + " at " + xpath,
			Err:      err,
		})
		return ""
	}
	if url != "" {
		slog.Info("Extracted attribute", slog.String("URL", url))
	}
	return url
}
