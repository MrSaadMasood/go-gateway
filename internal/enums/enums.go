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

var reqStatusMap = map[RequestStatus]RequestStatus{
	ReqInitialized:       ReqValidationSuccess,
	ReqValidationSuccess: ReqRateLimitSuccess,
	ReqRateLimitSuccess:  ReqProxySuccess,
	ReqProxySuccess:      ReqSuccess,
}

var reqStatus1Map = map[RequestStatus][]RequestStatus{
	ReqInitialized:       {ReqValidationSuccess, ReqValidationFailed},
	ReqValidationSuccess: {ReqRateLimitSuccess, ReqRateLimitFailed},
	ReqRateLimitSuccess:  {ReqProxySuccess, ReqProxyFailed},
	ReqProxySuccess:      {ReqSuccess, ReqFailed},
}

var ErrTerminalState = errors.New("final state reached")

func (*RequestStatus) Next(from RequestStatus) (RequestStatus, error) {
	if from == ReqSuccess || from == ReqFailed {
		return "", ErrTerminalState
	}

	state, ok := reqStatusMap[from]
	if !ok {
		return "", errors.New("invlalid state provided")
	}

	return state, nil
}

func (*RequestStatus) NextState(from RequestStatus, to RequestStatus) error {

	if from == ReqSuccess || from == ReqFailed {
		return ErrTerminalState
	}

	states, ok := reqStatus1Map[from]
	if !ok {
		return errors.New("invalid state provided")
	}

	allowed := slices.Contains(states, to)
	if !allowed {
		return errors.New("transition not allowed")
	}

	return nil
}
