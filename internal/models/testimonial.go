package models

import (
	"sync"

	"github.com/chromedp/cdproto/cdp"
)

type Testimonial struct {
	TestimonialWg   *sync.WaitGroup
	ImageWg         *sync.WaitGroup
	ImageResultChan chan []string
	Err             ErrorHandler
	TNodes          []*cdp.Node
}

type TestimonialResult struct {
	Name string
	Uri  string
}
