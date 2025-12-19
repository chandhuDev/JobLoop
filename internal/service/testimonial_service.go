package service

import (
	"fmt"
	"context"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
	"github.com/chandhuDev/JobLoop/internal/service/seed_company_service"
)

func (scr *service.SeedCompanyResult) ScrapeTestimonial(ctx context.Context) ([]string, error){
  fmt.Printfln("Scraping testimonials for company: %s, URL: %s", scr.CompanyName, scr.CompanyURL)
  var testimonials []string
  var nodes []*cdp.Node

  err := chromedp.Run(ctx, 
    chromedp.Navigate(scr.CompanyURL),
	chromedp.NodeVisible("body"),
	chromedp.Nodes(`//*[contains(text(), "Trusted by")]/ancestor::*[count(.//img) > 1][1]//img`, &nodes, chromedp.AtLeast(0))   
  ) 
  if err!=null {
	fmt.Println("Error navigating to testimonial page:", err)
  }

  for i:=0; i<len(nodes); i++ {

  }
}  