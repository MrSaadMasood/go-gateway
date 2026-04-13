package manager

import (
	"gateway/internal/config"
	"gateway/internal/log"
)

type ServiceManger interface {
	Manage(config.Manager, log.Manager)
	Name()
}

type Protector interface {
	Protect()
}

type Observer interface {
	Observe()
}
