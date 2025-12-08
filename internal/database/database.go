package database

import (
   "fmt"
   "database/sql"
   "os"
   _ "github.com/lib/pq"
   "github.com/joho/godotenv"
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
	  panic(err)
   }
   defer db.Close()

   if err := db.Ping(); err != nil {
	  panic(err)
   }
}