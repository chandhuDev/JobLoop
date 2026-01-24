package models

type NamesClient struct {
	NamesChan chan string
}

type LinkData struct {
	URL  string
	Text string
}
