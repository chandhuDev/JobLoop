package repository

import (
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func BulkUpsertTestimonials(
	db *gorm.DB,
	seedID uint,
	names []string,
) error {

	var records []schema.TestimonialCompany

	for _, name := range names {
		records = append(records, schema.TestimonialCompany{
			SeedCompanyID: seedID,
			CompanyName:   name,
		})
	}

	return db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "companyName"},
		},
		DoNothing: true,
	}).Create(&records).Error
}
