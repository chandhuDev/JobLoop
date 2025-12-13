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

func ConnectDatabase() error {
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
        return fmt.Errorf("db connect failed: %w", err)
    }

    DB = db
    return nil
}

func CreateSchema() error {
   err := DB.AutoMigrate(&schema.SeedCompanies{}, &schema.TestimonialCompanies{})
   return err
}