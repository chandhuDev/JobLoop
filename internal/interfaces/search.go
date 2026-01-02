package interfaces

type SearchClient interface {
	SearchKeyWordInGoogle(keyword string, i int, key string) (string, error)
}
