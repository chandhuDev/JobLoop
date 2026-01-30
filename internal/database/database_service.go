package database

import (
	"fmt"
	"os"

	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
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
		Logger: gormlogger.Default.LogMode(gormlogger.Info),
	})
	if err != nil {
		logger.Error().Err(err).Msg("db connect failed")
	}

	return &models.Database{
		DB: dbInstance,
	}
}

func (db *DatabaseService) CreateSchema() error {
	if err := db.DB.DB.Exec("CREATE EXTENSION IF NOT EXISTS citext").Error; err != nil {
		return fmt.Errorf("failed to create citext extension: %w", err)
	}
	err := db.DB.DB.AutoMigrate(&schema.SeedCompany{}, &schema.TestimonialCompany{}, &schema.Job{})
	return err
}

func (db *DatabaseService) GetDB() *gorm.DB {
	return db.DB.DB
}

func (db *DatabaseService) Close() error {
	sqlDB, err := db.DB.DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get sqlDB: %w", err)
	}
	return sqlDB.Close()
}
