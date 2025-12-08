package database

import (
   "fmt"
   "database/sql"
   _ "github.com/lib/pq"
)

func ConnectDatabase() {
   db, err := sql.Open("postgress", "user=postgres password=postgres dbname=jobloop")
   if err != nil {
	 panic(err)
   }
   defer db.Close()
   if err := db.Ping(); err != nil {
	panic(err)
}
}