package service

import (
	"context"
	"log/slog"
	"sync"

	interfaces "github.com/chandhuDev/JobLoop/internal/interfaces"
	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chromedp/chromedp"
)

type TestimonialService struct {
	Testimonial *models.Testimonial
}

func NewTestimonial() *models.Testimonial {
	return &models.Testimonial{
		ImageResultChan: make(chan []string, 100),
		TestimonialWg:   &sync.WaitGroup{},
		ImageWg:         &sync.WaitGroup{},
	}
}

func (t *TestimonialService) ScrapeTestimonial(scraper *interfaces.ScraperClient, scChan <-chan models.SeedCompanyResult, vision VisionWrapper) {

	for i := 0; i < 5; i++ {
		t.Testimonial.TestimonialWg.Add(1)

		go func(i int, browser interfaces.BrowserClient, scChan <-chan models.SeedCompanyResult, wg *sync.WaitGroup, im chan []string, e interfaces.ErrorClient) {

			tabContext, tabCancel := scraper.Browser.RunInNewTab()
			defer tabCancel()
			defer t.Testimonial.TestimonialWg.Done()
			xpath := `
		  (
			//*[contains(translate(text(), 'ABCDEFGHIJKLMNOPQRSTUVWXYZ', 'abcdefghijklmnopqrstuvwxyz'), 'trust')]
			  /following::*[count(.//img) >= 3][1]
		  )//img
		  `
			slog.Info("Starting Testimonial processor goroutine", slog.Int("goroutine id", i))

			for scr := range scChan {

				err := chromedp.Run(tabContext,
					chromedp.Navigate(scr.CompanyURL),
					chromedp.WaitVisible("body"),
					chromedp.Nodes(xpath, &t.Testimonial.TNodes, chromedp.BySearch, chromedp.AtLeast(0)),
				)
				if err != nil {
					e.Send(models.WorkerError{
						WorkerId: i,
						Message:  "Error navigating to testimonial page:",
						Err:      err,
					})
				}
				if len(t.Testimonial.TNodes) == 0 || t.Testimonial.TNodes == nil {
					slog.Info("No testimonial images found for", slog.String("company Name", scr.CompanyName))
					continue
				}
				var UrlArray []string

				for j := range t.Testimonial.TNodes {
					var fullURL string
					fullURL = getAttr(tabContext, t.Testimonial.TNodes[j].FullXPath(), "src", e)
					if fullURL == "" || fullURL == "null" {
						fullURL = getAttr(tabContext, t.Testimonial.TNodes[j].FullXPath(), "data-src", e)
					}
					UrlArray = append(UrlArray, fullURL)
				}
				im <- UrlArray
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")
				slog.Info("successfully appended")

			}

		}(i, scraper.Browser, scChan, t.Testimonial.TestimonialWg, t.Testimonial.ImageResultChan, scraper.Err)
	}

	// dd := make(chan []models.TestimonialResult, 50)

	for i := 0; i < 5; i++ {
		t.Testimonial.ImageWg.Add(1)

		go func(workerID int) {
			defer t.Testimonial.ImageWg.Done()

			slog.Info("Started image processor goroutine",
				slog.Int("goroutine_id", workerID),
			)

			for urlArray := range t.Testimonial.ImageResultChan {
				slog.Info("In processed Images of testimonials")
				result := vision.ExtractImageFromText(urlArray, scraper.Err)

				slog.Info("Processed testimonial images",
					slog.Any("result", result),
				)
			}
		}(i)
	}

	t.Testimonial.TestimonialWg.Wait()
	close(t.Testimonial.ImageResultChan)
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
