package common

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/enums"
	"gateway/internal/log"
	"regexp"
	"strings"
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

func WithLogData(ctx context.Context, ld *log.LogData) context.Context {
	return context.WithValue(ctx, "log-data", ld)
}

func LogDataFrom(ctx context.Context) (*log.LogData, error) {
	ld, ok := ctx.Value("log-data").(*log.LogData)
	if !ok {
		return nil, errors.New("no log data found")
	}
	return ld, nil
}

func RequestServiceExemptedPath(urlPath string) (path, version string, err error) {

	splitted := strings.SplitN(urlPath, "/", 4)
	if len(splitted) < 3 {
		return "", "", errors.New("req path not valid")
	}

	m, err := regexp.Match(`^v\d+$`, []byte(splitted[2]))
	if err != nil || m == false {
		return "", "", errors.New("version verification failed")
	}

	version = splitted[2]
	path = "/"
	if len(splitted) == 4 {
		path = splitted[3]
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
		}
	}

	return path, version, nil

}
