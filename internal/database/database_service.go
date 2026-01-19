package database

import (
	"fmt"
	"os"

	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type DatabaseService struct {
	DB *models.Database
}

func ConnectDatabase() *models.Database {
	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")

	dsn := fmt.Sprintf(
		"host=localhost user=%s password=%s dbname=jobloop sslmode=disable",
		user, pass,
	)

	dbInstance, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		fmt.Println("db connect failed: %w", err)
	}

	return &models.Database{
		DB:dbInstance}
}


func (db *DatabaseService) CreateSchema() error {
	err := db.DB.DB.AutoMigrate(&schema.SeedCompanies{}, &schema.TestimonialCompanies{})
	return err
}
