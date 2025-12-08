package main

import (
	"fmt"
	"github.com/chandhuDev/JobLoop/internal/browserscraper"
	"github.com/chandhuDev/JobLoop/internal/database"
)

func main() {
   fmt.Println("Starting JobLoop")
   if err := godotenv.Load(); err != nil {
	log.Println("No .env file found")
   }
   browserscraper.LaunchBrowser()
   database.ConnectDatabase()
}