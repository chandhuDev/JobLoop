package models

import (
	"sync"
)

type Testimonial struct {
	TestimonialWg   sync.WaitGroup
	ImageWg         sync.WaitGroup
	ImageResultChan chan []string
	Err             ErrorHandler
}

type TestimonialResult struct {
	Name string
	Uri  string
}
