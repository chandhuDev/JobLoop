package service

import (
	"fmt"
	"github.com/chromedp/chromedp"
	"github.com/chromedp/cdproto/cdp"
  "github.com/chandhuDev/JobLoop/internal/browser"
  "github.com/chandhuDev/JobLoop/internal/config/vision"
  visionApi "cloud.google.com/go/vision/apiv1"
  "sync"
)

func ScrapeTestimonial(browser browser.Browser, vision visionApi.ImageAnnotatorClient, seedCompanyList []SeedCompanyResult) {
  var testimonials []string
  var nodes []*cdp.Node
  var wg sync.WaitGroup
  ch := make(chan string)

  for i:=0; i<len(seedCompanyList); i++ {
    tabContext, tabCancel := browser.RunInNewTab()
    defer tabCancel()
    err := chromedp.Run(tabContext, 
    chromedp.Navigate(seedCompanyList[i].CompanyURL),
    chromedp.NodeVisible("body"),
    chromedp.Nodes(`//*[contains(text(), "Trusted by")]/ancestor::*[count(.//img) > 1][1]//img`, &nodes, chromedp.AtLeast(0)),   
    ) 
    if err!=nil {
    fmt.Println("Error navigating to testimonial page:", err)
    }
    for i:=0; i<len(nodes); i++ {

      go func(imageUrl string){
         imageData := extractImageData(imageUrl)
         ch <- imageData
      }(nodes[i])
  
      go func() {
        testimonialImageData := <- ch
        testimonialName := vision.ExtractImageFromText(testimonialImageData)
      }()
    }
  }
} 


func extractImageData(ImageUrl string)  {
  
}