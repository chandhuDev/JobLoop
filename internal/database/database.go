package database

import (
   "fmt"
   "database/sql"
   "os"
   _ "github.com/lib/pq"
)

func ConnectDatabase() {
   user := os.Getenv("DB_USER")
   pass := os.Getenv("DB_PASS")

   databaseOptions := fmt.Sprintf(
		"user=%s password=%s dbname=jobloop sslmode=disable",
		user, pass,
	)

   db, err := sql.Open("postgres", databaseOptions)

   if err != nil {
      fmt.Printf("DB connect failed: %v — continuing in debug mode\n", err)
   }
   defer db.Close()

   if err := db.Ping(); err != nil {
	   fmt.Printf("failed to ping the server", err)
   
   }
}