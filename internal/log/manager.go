package log

import "gateway/internal/config"

type Manager interface {
	Manage(config.Manager, Logger)
}
