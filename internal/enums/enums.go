package enums

import (
	"errors"
	"slices"
)

type ServiceStatus string

const (
	ServiceAvailable   ServiceStatus = "available"
	ServiceUnavailable ServiceStatus = "unavailable"
)

type RequestStatus string

const (
	ReqInitialized       RequestStatus = "request_initialized"
	ReqServiceMapSuccess RequestStatus = "request_service_map_success"
	ReqServiceMapFailed  RequestStatus = "request_service_map_failed"
	ReqValidationSuccess RequestStatus = "validation_success"
	ReqValidationFailed  RequestStatus = "validation_failed"
	ReqRateLimitSuccess  RequestStatus = "rate_limit_success"
	ReqRateLimitFailed   RequestStatus = "rate_limit_failed"
	ReqProxySuccess      RequestStatus = "proxy_success"
	ReqProxyFailed       RequestStatus = "proxy_failed"
	ReqSuccess           RequestStatus = "success"
	ReqFailed            RequestStatus = "failed"
	ReqTimeout           RequestStatus = "timeout"
)

var reqStatusMap = map[RequestStatus][]RequestStatus{
	ReqInitialized:       {ReqServiceMapSuccess, ReqServiceMapFailed, ReqTimeout},
	ReqServiceMapSuccess: {ReqValidationSuccess, ReqValidationFailed, ReqTimeout},
	ReqValidationSuccess: {ReqRateLimitSuccess, ReqRateLimitFailed, ReqTimeout},
	ReqRateLimitSuccess:  {ReqProxySuccess, ReqProxyFailed, ReqTimeout},
	ReqProxySuccess:      {ReqSuccess, ReqFailed, ReqTimeout},
	ReqServiceMapFailed:  {},
	ReqValidationFailed:  {},
	ReqRateLimitFailed:   {},
	ReqProxyFailed:       {},
	ReqSuccess:           {},
	ReqFailed:            {},
	ReqTimeout:           {},
}

var ErrReqTerminalState = errors.New("final state reached")

func (*RequestStatus) Next(from RequestStatus, to RequestStatus) error {

	if from == to {
		return errors.New("current and next states cannot be equal")
	}

	currStates, ok := reqStatusMap[from]
	if !ok {
		return errors.New("invalid state provided")
	}

	if len(currStates) == 0 || to == ReqInitialized {
		return ErrReqTerminalState
	}

	allowed := slices.Contains(currStates, to)
	if !allowed {
		return errors.New("transition not allowed")
	}

	return nil
}

func (rs RequestStatus) IsTerminalState(s RequestStatus) bool {
	terminalStates := make([]RequestStatus, 0)
	for k, v := range reqStatusMap {
		if len(v) == 0 {
			terminalStates = append(terminalStates, k)
		}
	}

	if slices.Contains(terminalStates, s) {
		return true
	}

	return false
}
