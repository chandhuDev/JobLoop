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
	mux.HandleFunc("GET /health", h.healthCheck)

	mux.HandleFunc("GET /api/companies", h.getCompanies)

	mux.HandleFunc("GET /api/jobs", h.getJobs)

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
	titleFilter := r.URL.Query().Get("title")

	filterType := r.URL.Query().Get("type")

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
	query := h.DB.Order("id DESC").Limit(limit).Offset(offset)

	switch filterType {
	case "engineering":
		query = query.Where("job_type = ?", "engineering")
	case "other":
		query = query.Where("job_type = ?", "other")
	case "all", "":
	default:
	}

	if titleFilter != "" {
		query = query.Where("job_title ILIKE ?", "%"+titleFilter+"%")
	}

	result := query.Find(&jobs)
	if result.Error != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to fetch jobs")
		return
	}

	var total int64
	countQuery := h.DB.Model(&schema.Job{})

	switch filterType {
	case "engineering":
		countQuery = countQuery.Where("job_type = ?", "engineering")
	case "other":
		countQuery = countQuery.Where("job_type = ?", "other")
	}

	if titleFilter != "" {
		countQuery = countQuery.Where("job_title ILIKE ?", "%"+titleFilter+"%")
	}
	countQuery.Count(&total)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
		"filter": filterType,
	})
}

func (h *Handlers) getDBStats(w http.ResponseWriter, r *http.Request) {
	var stats struct {
		Engineering int64 `json:"engineering"`
		Other       int64 `json:"other"`
		Noise       int64 `json:"noise"`
		Total       int64 `json:"total"`
	}

	h.DB.Model(&schema.Job{}).Where("job_type = ?", "engineering").Count(&stats.Engineering)
	h.DB.Model(&schema.Job{}).Where("job_type = ?", "other").Count(&stats.Other)
	h.DB.Model(&schema.Noise{}).Count(&stats.Noise)
	stats.Total = stats.Engineering + stats.Other

	h.jsonResponse(w, http.StatusOK, stats)
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
