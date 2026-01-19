package interfaces


type DatabaseClient interface {
	CreateSchema() error
}
