package schema

import (
    "time"
)

type SeedCompanies struct {
	ID  uint `gorm:"primaryKey"`
	CompanyName string 
	CompanyURL string
	Visited bool
	Time time.Time
	TestimonialCompanies []TestimonialCompanies `gorm:"foreignKey:SeedCompanyID"`
}

type TestimonialCompanies struct {
	ID  uint `gorm:"primaryKey"`
	SeedCompanyID uint
	CompanyName string
}