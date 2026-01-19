package interfaces

import (
	vision "cloud.google.com/go/vision/apiv1"
)

type ScraperClient struct {
	Browser BrowserClient
	Search  SearchClient
	Vision  *vision.ImageAnnotatorClient
	Err     ErrorClient
	DbClient DatabaseClient
}
