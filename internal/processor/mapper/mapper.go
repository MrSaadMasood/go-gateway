package mapper

import "gateway/internal/data"

type JobDefinitionMap map[string]data.JobExecutor

type JobMapper interface {
	Map(jobName string) (data.JobExecutor, error)
}
