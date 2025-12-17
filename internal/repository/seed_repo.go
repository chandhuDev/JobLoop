package repository

import (
	"gorm.io/gorm"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"time"
)


func CreateSeedCompanyRepository(CompanyName string, CompanyURL string, TestimonialArray []schema.TestimonialCompanies) schema.SeedCompanies {
	return schema.SeedCompanies{
		CompanyName: CompanyName,
		CompanyURL:  CompanyURL,
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