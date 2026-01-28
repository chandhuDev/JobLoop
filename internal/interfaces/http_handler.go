package interfaces

type HTTPHandlerInterface interface {
    RegisterRoutes()
	HealthCheck()
	GetCompanies(db *DatabaseClient)
	GetJobs(db *DatabaseClient)
	GetDBState(db *DatabaseClient)
}