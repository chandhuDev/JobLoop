package repository

import (
	"log/slog"

	models "github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
)

func CreateSeedCompanyRepository(cd models.SeedCompanyResult, TestimonialArray []schema.TestimonialCompany) schema.SeedCompany {
	return schema.SeedCompany{
		CompanyName: cd.CompanyName,
		CompanyURL:  cd.CompanyURL,
	}
}

func CreateSeedCompany(seedCompany schema.SeedCompany, DB *gorm.DB) error {
	result := DB.Create(&seedCompany)
	if result.Error != nil {
		return result.Error
	}
	slog.Info("Seed company created successfully", slog.String("CompanyName", seedCompany.CompanyName))
	return nil
}
