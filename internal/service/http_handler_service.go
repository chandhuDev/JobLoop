package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	models "github.com/chandhuDev/JobLoop/internal/models"
)

type HTTPHandlerService struct {
	HttpHandler *models.HTTPHandler
}

func NewHTTPHandlerService(db *models.Database) *HTTPHandlerService {
	h := &HTTPHandlerService{
		HttpHandler: &models.HTTPHandler{
			Db:        db,
			ServerMux: http.NewServeMux(),
		},
	}

	h.registerRoutes()
	return h
}

func (h *HTTPHandlerService) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.HttpHandler.ServerMux.ServeHTTP(w, r)
}

func (h *HTTPHandlerService) registerRoutes() {
	// Health check
	h.HttpHandler.ServerMux.HandleFunc("GET /health", h.healthCheck)

	// Companies
	h.HttpHandler.ServerMux.HandleFunc("GET /api/companies", h.getCompanies)

	// Jobs
	h.HttpHandler.ServerMux.HandleFunc("GET /api/jobs", h.getJobs)

	// Stats
	h.HttpHandler.ServerMux.HandleFunc("GET /api/state", h.getDBStats)
}

func (h *HTTPHandlerService) healthCheck(w http.ResponseWriter, r *http.Request) {
	h.jsonResponse(w, http.StatusOK, map[string]string{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func (h *HTTPHandlerService) getCompanies(w http.ResponseWriter, r *http.Request) {
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

	// Query database
	var companies []models.SeedCompany
	result := h.HttpHandler.Db.DB.Limit(limit).Offset(offset).Find(&companies)
	if result.Error != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to fetch companies")
		return
	}
	var total int64
	h.HttpHandler.Db.DB.Model(&models.SeedCompany{}).Count(&total)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":   companies,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})

}

func (h *HTTPHandlerService) getJobs(w http.ResponseWriter, r *http.Request) {
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

	var jobs []models.Job
	query := h.HttpHandler.Db.DB.Limit(limit).Offset(offset)

	if companyID != "" {
		query = query.Where("company_id = ?", companyID)
	}
	result := query.Find(&jobs)
	if result.Error != nil {
		h.errorResponse(w, http.StatusInternalServerError, "Failed to fetch jobs")
		return
	}

	var total int64
	countQuery := h.HttpHandler.Db.DB.Model(&models.Job{})
	if companyID != "" {
		countQuery = countQuery.Where("company_id = ?", companyID)
	}
	countQuery.Count(&total)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"data":   jobs,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

func (h *HTTPHandlerService) getDBStats(w http.ResponseWriter, r *http.Request) {
	var companyCount int64
	var jobCount int64

	h.HttpHandler.Db.DB.Model(&models.SeedCompany{}).Count(&companyCount)
	h.HttpHandler.Db.DB.Model(&models.Job{}).Count(&jobCount)

	h.jsonResponse(w, http.StatusOK, map[string]interface{}{
		"companies": companyCount,
		"jobs":      jobCount,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *HTTPHandlerService) jsonResponse(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandlerService) errorResponse(w http.ResponseWriter, status int, message string) {
	h.jsonResponse(w, status, map[string]string{"error": message})
}

func parseInt(s string) (int, error) {
	var i int
	_, err := fmt.Sscanf(s, "%d", &i)
	return i, err
}
