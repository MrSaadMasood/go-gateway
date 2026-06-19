package common

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/enums"
)

func WithMappedService(ctx context.Context, s config.ServiceConfig) context.Context {
	return context.WithValue(ctx, "mapped-service", s)
}

func MappedServiceFrom(ctx context.Context) (config.ServiceConfig, error) {
	s, ok := ctx.Value("mapped-service").(config.ServiceConfig)
	if !ok {
		return config.ServiceConfig{}, errors.New("req not mapped with service")
	}
	return s, nil
}

func WithReqStatus(ctx context.Context, status enums.RequestStatus) context.Context {
	return context.WithValue(ctx, "status", status)
}

func StatusFrom(ctx context.Context) (enums.RequestStatus, error) {
	s, ok := ctx.Value("status").(enums.RequestStatus)
	if !ok {
		return "", errors.New("status not found from context")
	}
	return s, nil
}
