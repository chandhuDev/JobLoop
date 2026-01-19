package schema

import (
    "time"
	"gorm.io/gorm"
)

type SeedCompanies struct {
	ID  uint `gorm:"primaryKey"`
	CompanyName string `gorm:"index"`
	CompanyURL string 
	Visited bool
	Time time.Time
	Status string
	TestimonialCompanies []TestimonialCompanies `gorm:"foreignKey:SeedCompanyID"`
}

type TestimonialCompanies struct {
	ID  uint `gorm:"primaryKey"`
	SeedCompanyID uint
	CompanyName string `gorm:"index"`
}

type DatabaseRepository struct {
	DB *gorm.DB
}
