package enums

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

type ReqEvent string

const (
	InitializeReqEvent        ReqEvent = "initialize_request_event"
	MapSuccessReqEvent        ReqEvent = "map_success_request_event"
	MapFailedReqEvent         ReqEvent = "map_failed_request_event"
	ValidationSuccessReqEvent ReqEvent = "validation_success_request_event"
	ValidationFailedReqEvent  ReqEvent = "validation_failed_request_event"
	RateLimitSuccessReqEvent  ReqEvent = "rate_limit_success_request_event"
	RateLimitFailedReqEvent   ReqEvent = "rate_limit_failed_request_event"
	ProxySuccessReqEvent      ReqEvent = "proxy_success_request_event"
	ProxyFailedReqEvent       ReqEvent = "proxy_failed_request_event"
	ReqSuccessEvent           ReqEvent = "request_success_event"
	ReqFailedEvent            ReqEvent = "request_failed_event"
	ReqTimeoutEvent           ReqEvent = "request_timeout_event"
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
