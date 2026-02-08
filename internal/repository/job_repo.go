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
	engineeringPatterns = []string{
		"engineer", "developer", "software", "backend", "frontend",
		"full stack", "fullstack", "devops", "sre", "data engineer",
		"machine learning", "ml engineer", "ai engineer", "architect",
		"programming", "coding", "technical lead", "tech lead",
		"qa engineer", "test engineer", "platform engineer",
		"infrastructure", "security engineer", "cloud engineer",
		"golang", "python", "java", "javascript", "react", "node",
		"ios", "android", "mobile", "embedded", "systems engineer",
	}

	noisePatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)create\s+account`),
		regexp.MustCompile(`(?i)sign\s+up`),
		regexp.MustCompile(`(?i)log\s*in`),
		regexp.MustCompile(`(?i)getting\s+started`),
		regexp.MustCompile(`(?i)privacy\s+policy`),
		regexp.MustCompile(`(?i)cookie\s+policy`),
		regexp.MustCompile(`(?i)terms\s+(of\s+)?(service|use)`),
		regexp.MustCompile(`help\..*\.com`),
		regexp.MustCompile(`support\..*\.com`),
		regexp.MustCompile(`(?i)contact\s+us`),
		regexp.MustCompile(`(?i)^\s*faq\s*$`),
		regexp.MustCompile(`(?i)medium\.com`),	
	}

	excludeTitles = []string{
		"manager", "sales", "marketing", "recruiter",
		"hr", "human resources", "customer success",
		"support specialist", "coordinator", "representative",
	}
)

func isNoise(title string) bool {
	for _, pattern := range noisePatterns {
		if pattern.MatchString(title) {
			return true
		}
	}
	return len(strings.TrimSpace(title)) < 3
}

func hasEngineeringKeyword(text string) bool {
	for _, keyword := range engineeringPatterns {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}

func isEngineeringJob(title string) bool {
	titleLower := strings.ToLower(title)

	if isNoise(title) {
		return false
	}

	for _, excluded := range excludeTitles {
		if strings.Contains(titleLower, excluded) {
			if !hasEngineeringKeyword(titleLower) {
				return false
			}
		}
	}

	return hasEngineeringKeyword(titleLower)
}

func UpsertJob(DB *gorm.DB, scid uint, jobs []models.LinkData) error {
	var jobRecords []schema.Job
	var noiseRecords []schema.Noise

	engineeringCount := 0
	noiseCount := 0
	otherCount := 0

	for _, job := range jobs {
		if isNoise(job.Text) {
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
			jobRecords = append(jobRecords, schema.Job{
				SeedCompanyID: scid,
				JobTitle:      job.Text,
				JobUrl:        job.URL,
				IsEngineering: isEng,
				JobType:       jobType,
			})
		} else {
			jobType = "other"
			otherCount++
			jobRecords = append(jobRecords, schema.Job{
				SeedCompanyID: scid,
				JobTitle:      job.Text,
				JobUrl:        job.URL,
				IsEngineering: false,
				JobType:       jobType,
			})
		}
	}

	logger.Info().Int("engineering_jobs", engineeringCount).Int("other_jobs", otherCount).Int("noise_filtered", noiseCount).Uint("seed_company_id", scid).Msg("upserting jobs")

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
