package interfaces

type SearchClient interface {
	SearchKeyword(companyName string, workerId int) (string, error)
}
