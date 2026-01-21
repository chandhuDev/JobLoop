package repository

import (
	"fmt"
	"log/slog"

	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
)

func CreateSeedCompanyRepository(scn string, scu string) schema.SeedCompany {
	return schema.SeedCompany{
		CompanyName: scn,
		CompanyURL:  scu,
		Visited:     true,
	}
}

func UpdateSeedCompanyData(scid uint, DB *gorm.DB, flags map[string]interface{},
) error {
	allowed := map[string]bool{
		"TestimonialScraped": true,
		"JobScraped":         true,
	}
	for key := range flags {
		if !allowed[key] {
			return fmt.Errorf("invalid property: %s", key)
		}
	}
	return DB.Model(&schema.SeedCompany{}).
		Where("id = ?", scid).
		Updates(flags).
		Error
}

func CreateSeedCompany(seedCompany schema.SeedCompany, DB *gorm.DB) error {
	result := DB.Create(&seedCompany)
	if result.Error != nil {
		return result.Error
	}
	slog.Info("Seed company created successfully", slog.String("CompanyName", seedCompany.CompanyName))
	return nil
}
