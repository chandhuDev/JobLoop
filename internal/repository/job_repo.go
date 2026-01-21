package repository

import (
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func UpsertJob(DB *gorm.DB, scid uint, jobs []string) error {
	var jobRecords []schema.Job
	for _, job := range jobs {
		jobRecords = append(jobRecords, schema.Job{
			SeedCompanyID: scid,
			JobTitle:      job,
		})
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "seed_company_id"},
			{Name: "job_title"},
		},
		DoNothing: true,
	}).Create(&jobRecords).Error
}
