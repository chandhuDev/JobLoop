package models

import (
	"google.golang.org/api/customsearch/v1"
)

type Search struct {
	SearchClient *customsearch.Service
}
