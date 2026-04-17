package auditor

import "time"

type FailureData struct {
	Reason string
}

type JobResult struct {
	JobId string
	State string
	*FailureData
}

type LogData struct {
	Source    string
	CreatedAt time.Time
	JobResult
}

type Auditor interface {
	Log(LogData)
}
