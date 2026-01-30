package models

import "net/http"

type HTTPHandler struct {
	Db *Database
	ServerMux *http.ServeMux
}