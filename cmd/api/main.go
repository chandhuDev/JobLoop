package main

import (
	"fmt"
	"time"

	"github.com/chandhuDev/JobLoop/internal/browser"

	// "github.com/chandhuDev/JobLoop/internal/database"
	"context"
	"log"

	"github.com/chandhuDev/JobLoop/internal/config/search"
	"github.com/chandhuDev/JobLoop/internal/config/vision"
	"github.com/chandhuDev/JobLoop/internal/service"
	"github.com/joho/godotenv"
)

func main() {
	fmt.Println("Starting JobLoop")
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	browserOptions := browser.Options{
		Disbale_gpu:  true,
		WindowWidth:  1920,
		WindowHeight: 1080,
	}
	browser := browser.CreateNewBrowser(browserOptions)
	defer browser.Close()

	SeedCompanyConfigs := []service.SeedCompanyConfig{
		//   {
		//    Name: "Y Combinator",
		//    URL: "http://www.ycombinator.com/companies",
		//    Selector: `span[class^="_coName_i9oky_470"]`,
		//    WaitTime: 10 * time.Second,
		//   },
		{
			Name:     "Peer list",
			URL:      "https://peerlist.io/jobs",
			Selector: `a[href^="/company/"][href*="/careers/"]`,
			WaitTime: 5 * time.Second,
		},
	}
	customSearchInstance := search.CreateSearchService(context.Background())

	seedCompanyResultArray := service.SeedCompanyConfigs(browser, SeedCompanyConfigs, customSearchInstance)
	fmt.Println("Seed Company Results:", seedCompanyResultArray)

	visionInstance, _ := vision.CreateVisionInit(context.Background())
	defer visionInstance.Close()
	vision := service.SetUpVision(visionInstance)
	service.ScrapeTestimonial(browser, *vision, seedCompanyResultArray)

	// db := database.ConnectDatabase()
	// if err := database.CreateSchema(); err!= nil {
	//    log.Fatalf("Failed to create schema: %v", err)
	// }
	// _ := database.SetUpDatabase(db)
}
