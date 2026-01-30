package interfaces

type HTTPHandlerInterface interface {
    RegisterRoutes()
	healthCheck()
	getCompanies()
	getJobs()
	getDBStats()
	jsonResponse()
	errorResponse()
	ServeHTTP()
}