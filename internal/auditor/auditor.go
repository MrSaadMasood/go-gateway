package auditor

import "time"

type LogData struct {
	source    string
	createdAt time.Time
	data      any
}

type Auditor interface {
	Log(LogData)
}
