package interfaces

type Job interface {
	InsertJobs(SeedCompanyId uint, jobs []string) error
}
