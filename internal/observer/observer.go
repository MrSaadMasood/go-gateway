package observer

import "time"

type ServiceOptions struct {
	ServiceName         string
	HealthCheckInterval *time.Time
}

type ServiceObserver interface {
	Observe([]ServiceOptions) error
}
