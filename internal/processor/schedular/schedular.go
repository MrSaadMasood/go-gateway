package schedular

import "gateway/internal/data"

type JobQueue []data.JobData

type Schedular interface {
	Schedule(data.JobData)
}
