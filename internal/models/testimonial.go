package models

import (
	"sync"
)

type Testimonial struct {
	TestimonialWg   *sync.WaitGroup
	ImageWg         *sync.WaitGroup
	ImageResultChan chan TestimonialImageResult
	Err             ErrorHandler
}

type TestimonialImageResult struct {
	SeedCompanyId uint
	CompanyName   string
	URL           []string
}

type TestimonialResult struct {
	Name string
}
