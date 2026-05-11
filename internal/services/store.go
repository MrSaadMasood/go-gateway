package services

import (
	"errors"
	"gateway/internal/config"
	"strings"
)

type Storer interface {
	Map(reqPath string) (config.ServiceConfig, error)
}

type ServicesStore struct {
	serviceConfigs []config.ServiceConfig
}

type GetServiceFunc func(scs []config.ServiceConfig) Storer

func NewMockServiceStore(scs []config.ServiceConfig) ServicesStore {
	return ServicesStore{
		serviceConfigs: scs,
	}
}

func (s ServicesStore) Map(path string) (config.ServiceConfig, error) {
	for _, v := range s.serviceConfigs {
		exists := strings.Contains(path, v.ServiceName)
		if exists {
			return v, nil
		}
	}
	return config.ServiceConfig{}, errors.Errorf("path does not contain the service")

}
