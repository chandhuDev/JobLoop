package handlers

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/chandhuDev/JobLoop/internal/schema"
	"gorm.io/gorm"
)

type Handlers struct {
	DB *gorm.DB
}

func NewHandlers(db *gorm.DB) *Handlers {
	return &Handlers{DB: db}
}

func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	// Health check
	mux.HandleFunc("GET /health", h.healthCheck)

	// Companies
	mux.HandleFunc("GET /api/companies", h.getCompanies)

	// Jobs
	mux.HandleFunc("GET /api/jobs", h.getJobs)

	// Stats
	mux.HandleFunc("GET /api/state", h.getDBStats)
}

func (h *Handlers) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) getCompanies(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var companies []schema.SeedCompany
	result := h.DB.Order("id ASC").Limit(limit).Offset(offset).Find(&companies)
	if result.Error != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to fetch companies")
		return
	}
	var total int64
	h.DB.Model(&schema.SeedCompany{}).Count(&total)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":   companies,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handlers) getJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	offset := 0
	companyID := r.URL.Query().Get("company_id")

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := parseInt(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := parseInt(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	var jobs []schema.Job
	query := h.DB.Order("id ASC").Limit(limit).Offset(offset)

	if companyID != "" {
		query = query.Where("seed_company_id = ?", companyID)
	}
	result := query.Find(&jobs)
	if result.Error != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to fetch jobs")
		return
	}

	var total int64
	countQuery := h.DB.Model(&schema.Job{})
	if companyID != "" {
		countQuery = countQuery.Where("seed_company_id = ?", companyID)
	}
	countQuery.Count(&total)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *Handlers) getDBStats(w http.ResponseWriter, r *http.Request) {
	var companyCount int64
	var jobCount int64

	h.DB.Model(&schema.SeedCompany{}).Count(&companyCount)
	h.DB.Model(&schema.Job{}).Count(&jobCount)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"companies": companyCount,
		"jobs":      jobCount,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *Handlers) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *Handlers) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
