package data

type JobState struct{ name string }

var (
	Queued  = JobState{name: "queueud"}
	Running = JobState{name: "running"}
	Success = JobState{name: "success"}
	Failed  = JobState{name: "failed"}
)

type JobData struct {
	Id      string
	JobName string
	Payload any
	State   JobState
}

type JobExecutor interface {
	Execute() error
}
