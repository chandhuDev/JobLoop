package main

import (
	"fmt"
   "time"
	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chandhuDev/JobLoop/internal/database"
	"github.com/chandhuDev/JobLoop/internal/service"
   "log"
   "github.com/joho/godotenv"
   "github.com/chandhuDev/JobLoop/internal/config/vision"
   "context"
)

func main() {
   fmt.Println("Starting JobLoop")
   if err := godotenv.Load(); err != nil {
	log.Println("No .env file found")
   }

   browserOptions := browser.Options{
      Headless : true,
      Disbale_gpu : true,
      WindowWidth : 1920,
      WindowHeight : 1080,
   }
   browser := browser.CreateNewBrowser(browserOptions)
   defer browser.Close()

   SeedCompanyConfigs := []service.SeedCompanyConfig{
     {
      Name: "Y Combinator",
      URL: "http://www.ycombinator.com/companies",
      Selector: "",
      WaitTime: 5 * time.Second,
     },
     {
      Name: "Peer list",
      URL: "https://peerlist.io/jobs",
      Selector: `a[href^="/company/"][href*="/careers/"]`,
      WaitTime: 5 * time.Second,
     },
   }
 
   seedCompanyResultArray := service.SeedCompanyConfigs(browser, SeedCompanyConfigs)
   fmt.Println("Seed Company Results:", seedCompanyResultArray)

   visionInstance, _ := vision.CreateVisionInit(context.Background())
   defer visionInstance.Close()
   vision := service.SetUpVision(visionInstance)
   service.ScrapeTestimonial(browser, *vision, seedCompanyResultArray)


   db := database.ConnectDatabase()
   if err := database.CreateSchema(); err!= nil {
      log.Fatalf("Failed to create schema: %v", err)
   }
   // _ := database.SetUpDatabase(db)
}