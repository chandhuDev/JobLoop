package main

import (
	"fmt"
   "time"
	"github.com/chandhuDev/JobLoop/internal/browser"
	"github.com/chandhuDev/JobLoop/internal/database"
	"github.com/chandhuDev/JobLoop/internal/service/seed_company_service"
   "log"
   "github.com/joho/godotenv"
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
   browser, err := browser.CreateNewBrowser(browserOptions)
   defer browser.Close()



   SeedCompanyConfig := seed_company_service.SeedCompanyConfig{
      Name : "Y Combinator",
      URL : "http://www.ycombinator.com/companies",
      Selector : "",
      WaitTime : 5 * time.Second,
   }
   seedCompanyScraper := seed_company_service.NewSeedCompanyScraper(SeedCompanyConfig)
   err := seedCompanyScraper.ScrapeSeedCompanies()




   db := database.ConnectDatabase(browserOptions)
   if err := database.CreateSchema(); err!= nil {
      log.Fatalf("Failed to create schema: %v", err)
   }
   _ := database.SetUpDatabase(db)
}