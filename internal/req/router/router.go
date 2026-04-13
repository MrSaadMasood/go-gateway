package router

import (
	"gateway/internal/config"
	"gateway/internal/req/proxy"
)

type Router interface {
	Map(config.Manager, proxy.Manager)
}
