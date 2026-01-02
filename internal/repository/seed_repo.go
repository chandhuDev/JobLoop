package repository

import (
	"log/slog"
	"time"

	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
)

func CreateSeedCompanyRepository(cd models.SeedCompanyResult, TestimonialArray []schema.TestimonialCompanies) schema.SeedCompanies {
	return schema.SeedCompanies{
		CompanyName:          cd.CompanyName,
		CompanyURL:           cd.CompanyURL,
		Visited:              false,
		Time:                 time.Now(),
		TestimonialCompanies: TestimonialArray,
	}
}

func CreateSeedCompany(seedCompany schema.SeedCompanies, DB *gorm.DB) error {
	result := DB.Create(&seedCompany)
	if result.Error != nil {
		return result.Error
	}
	slog.Info("Seed company created successfully", slog.String("CompanyName", seedCompany.CompanyName))
	return nil
}
