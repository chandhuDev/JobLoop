package repository

import (
	"gorm.io/gorm"
	"time"
	"github.com/chandhuDev/JobLoop/internal/schema"
)

funct CreateTestimonialCompanyRepository(SeedCompanyID uint, CompanyName string ) []schema.TestimonialCompanies {
	return []schema.TestimonialCompanies{
		SeedCompanyID: SeedCompanyID,
		CompanyName: CompanyName,
	}
}



