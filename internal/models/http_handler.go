package models

import "net/http"

type HTTPHandler struct {
	db Database
	server *http.ServeMux
}