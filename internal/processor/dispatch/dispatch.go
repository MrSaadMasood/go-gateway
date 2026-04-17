package dispatch

import "gateway/internal/data"

type Jobber struct {
}

type Dispatcher interface {
	Dispatch(data.JobData) error
}
