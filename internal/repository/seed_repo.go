package repository

import (
	"gorm.io/gorm"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"github.com/chandhuDev/JobLoop/internal/service/seed_company_service"
	"time"
)


func CreateSeedCompanyRepository(cd *seed_company_service.CompanyResult, TestimonialArray []schema.TestimonialCompanies) schema.SeedCompanies {
	return schema.SeedCompanies{
		CompanyName: cd.CompanyName,
		CompanyURL:  cd.CompanyURL,
		Visited:     false,
		Time:        time.Now(),
		TestimonialCompanies: TestimonialArray,
	}
}

func CreateSeedCompany(seedCompany schema.SeedCompanies, DB *gorm.DB) error{
   result := DB.Create(&seedCompany)
   if result.Error !=nil {
	return result.Error
   }
   fmt.Printf("%s rows inserted to table", result.RowsAffected)
   return nil
}