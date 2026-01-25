package repository

import (
	"fmt"
	"log/slog"

	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
)

func CreateSeedCompanyRepository(scn string, scu string) *schema.SeedCompany {
	return &schema.SeedCompany{
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

func CreateSeedCompany(seedCompany *schema.SeedCompany, DB *gorm.DB) error {
	var existing schema.SeedCompany
	existingResult := DB.Where("company_url = ?", seedCompany.CompanyURL).First(&existing)

	if existingResult.Error == nil {
		// Exists - update and return existing ID
		seedCompany.ID = existing.ID
		return DB.Model(&existing).Updates(map[string]interface{}{
			"visited":      seedCompany.Visited,
			"company_name": seedCompany.CompanyName,
		}).Error
	}

	// Doesn't exist - create new
	newRecordResult := DB.Create(seedCompany)
	if newRecordResult.Error != nil {
		return newRecordResult.Error
	}
	if seedCompany.ID == 0 {
		slog.Error("failed to create seed company, ID is zero", slog.String("company_url", seedCompany.CompanyURL))
		return fmt.Errorf("failed to create seed company, ID is zero")
	}
	return nil
}
