package repository

import (
	"fmt"
	"time"

	"github.com/chandhuDev/JobLoop/internal/schema"
	"github.com/chandhuDev/JobLoop/internal/service"
	"gorm.io/gorm"
)

func CreateSeedCompanyRepository(cd service.SeedCompanyResult, TestimonialArray []schema.TestimonialCompanies) schema.SeedCompanies {
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
	fmt.Printf("%s rows inserted to table", result.RowsAffected)
	return nil
}
