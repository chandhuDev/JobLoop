package repository

import (
	"github.com/chandhuDev/JobLoop/internal/schema"
)

func CreateTestimonialCompanyRepository(SeedCompanyID uint, CompanyName string) schema.TestimonialCompanies {
	return schema.TestimonialCompanies{
		SeedCompanyID: SeedCompanyID,
		CompanyName:   CompanyName,
	}
}
