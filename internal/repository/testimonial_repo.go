package repository

import (
	"github.com/chandhuDev/JobLoop/internal/schema"
)

func CreateTestimonialCompanyRepository(SeedCompanyID uint, CompanyName string) schema.TestimonialCompany {
	return schema.TestimonialCompany{
		SeedCompanyID: SeedCompanyID,
		CompanyName:   CompanyName,
	}
}
