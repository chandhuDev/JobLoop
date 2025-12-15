package database

import (
    "fmt"
    "os"
    "gorm.io/driver/postgres"
    "gorm.io/gorm"
    "github.com/chandhuDev/JobLoop/internal/schema"
    "gorm.io/gorm/logger"
)

var DB *gorm.DB

func ConnectDatabase() *gorm.DB {
    user := os.Getenv("DB_USER")
    pass := os.Getenv("DB_PASS")

    dsn := fmt.Sprintf(
        "host=localhost user=%s password=%s dbname=jobloop sslmode=disable",
         user, pass,
    )

    db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
      Logger: logger.Default.LogMode(logger.Info),
   })
    if err != nil {
        fmt.Printf("db connect failed: %w", err)
    }
    
    DB = db
    return db
}

func CreateSchema() error {
   err := DB.AutoMigrate(&schema.SeedCompanies{}, &schema.TestimonialCompanies{})
   return err
}