package repository

import (
	"regexp"
	"strings"

	"github.com/chandhuDev/JobLoop/internal/logger"
	"github.com/chandhuDev/JobLoop/internal/models"
	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	engineeringRegex = regexp.MustCompile(`(?i)(engineer|developer|software|backend|frontend|full[\s-]?stack|devops|sre|data engineer|machine learning|ml engineer|ai engineer|architect|programming|coding|technical lead|tech lead|qa engineer|test engineer|platform engineer|infrastructure|security engineer|cloud engineer|golang|python|java|javascript|react|node|ios|android|mobile|embedded|systems engineer)`)

	noiseRegex = regexp.MustCompile(`(?i)(create\s+account|sign\s+up|log\s*in|getting\s+started|privacy\s+policy|cookie\s+policy|terms\s+(of\s+)?(service|use)|contact\s+us|^\s*faq\s*$|medium\.com|help\..*\.com|support\..*\.com|linkedin\.com|partner-with-us|account|subscription|privacy|terms|product-help|product-support|clear-finance|customers?|customer-service|product-updates|-vs-|product[- ]?tour|business|product-marketing|changelog|mergers-and-acquisitions|customer-experience|explorer|defence|blog|ads?|reviews?|case-studies?|partners?|sales|business-funding|schedule-a-call)`)

	excludeTitles = []string{
		"manager", "sales", "marketing", "recruiter",
		"hr", "human resources", "customer success",
		"support specialist", "coordinator", "representative",
	}
)

func isNoise(title, url string) bool {
	if len(strings.TrimSpace(title)) < 3 {
		return true
	}

	return noiseRegex.MatchString(title) || noiseRegex.MatchString(url)
}

func isEngineeringJob(title string) bool {
	titleLower := strings.ToLower(title)

	if noiseRegex.MatchString(titleLower) {
		return false
	}

	for _, excluded := range excludeTitles {
		if strings.Contains(titleLower, excluded) && !engineeringRegex.MatchString(titleLower) {
			return false
		}
	}

	return engineeringRegex.MatchString(titleLower)
}

func UpsertJob(DB *gorm.DB, scid uint, jobs []models.LinkData) error {
	var jobRecords []schema.Job
	var noiseRecords []schema.Noise

	engineeringCount := 0
	noiseCount := 0
	otherCount := 0

	for _, job := range jobs {

		if isNoise(job.Text, job.URL) {
			noiseRecords = append(noiseRecords, schema.Noise{
				NoiseUrl:      job.URL,
				NoiseText:     job.Text,
				SeedCompanyID: scid,
			})
			noiseCount++
			continue
		}

		isEng := isEngineeringJob(job.Text)
		jobType := "other"

		if isEng {
			jobType = "engineering"
			engineeringCount++
		} else {
			otherCount++
		}

		jobRecords = append(jobRecords, schema.Job{
			SeedCompanyID: scid,
			JobTitle:      job.Text,
			JobUrl:        job.URL,
			IsEngineering: isEng,
			JobType:       jobType,
		})
	}

	logger.Info().
		Int("engineering_jobs", engineeringCount).
		Int("other_jobs", otherCount).
		Int("noise_filtered", noiseCount).
		Uint("seed_company_id", scid).
		Msg("upserting jobs")

	if len(noiseRecords) > 0 {
		if err := DB.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "noise_url"}},
			DoNothing: true,
		}).Create(&noiseRecords).Error; err != nil {
			logger.Error().Err(err).Msg("failed to insert noise records")
		}
	}

	if len(jobRecords) == 0 {
		logger.Warn().Uint("seed_company_id", scid).Msg("no valid jobs to insert after filtering")
		return nil
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "seed_company_id"},
			{Name: "job_title"},
		},
		DoNothing: true,
	}).Create(&jobRecords).Error
}
