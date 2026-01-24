package schema

import (
	"time"
)

type SeedCompany struct {
	ID uint `gorm:"primaryKey"`

	CompanyName string `gorm:"not null;uniqueIndex"`
	CompanyURL  string `gorm:"not null;uniqueIndex"`

	Visited            bool `gorm:"default:false"`
	TestimonialScraped bool `gorm:"default:false"`
	JobScraped         bool `gorm:"default:false"`

	CreatedAt time.Time

	Testimonials []TestimonialCompany `gorm:"constraint:OnDelete:CASCADE;foreignKey:SeedCompanyID"`
	Jobs         []Job                `gorm:"constraint:OnDelete:CASCADE;foreignKey:SeedCompanyID"`
}

type TestimonialCompany struct {
	ID uint `gorm:"primaryKey"`

	SeedCompanyID uint   `gorm:"not null;index"`
	CompanyName   string `gorm:"type:citext;not null;uniqueIndex"`

	CreatedAt time.Time
}

type Job struct {
	ID uint `gorm:"primaryKey"`

	SeedCompanyID uint   `gorm:"not null;uniqueIndex:uniq_job_index"`
	JobTitle      string `gorm:"type:citext;not null;uniqueIndex:uniq_job_index"`
	JobUrl        string `gorm:"not null"`

	CreatedAt time.Time
}
