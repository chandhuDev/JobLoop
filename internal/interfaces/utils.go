package interfaces

type UtilsScraper interface {
	ReturnNamesChan() chan string
}