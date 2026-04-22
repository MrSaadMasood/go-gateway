package services

import "gateway/internal/config"

type Storer interface {
	Map(reqPath string) (config.ServiceConfig, error)
}
