package enums

type ServiceStatus string

const (
	ServiceAvailable   ServiceStatus = "available"
	ServiceUnavailable ServiceStatus = "unavailable"
)

type RequestStatus string

const (
	ReqSuccess RequestStatus = "success"
	ReqFailed  RequestStatus = "failed"
	ReqTimeout RequestStatus = "timeout"
)

func (rs *RequestStatus) Success() RequestStatus {
	return ReqSuccess
}

func (rs *RequestStatus) Failed() RequestStatus {
	return ReqFailed
}

func (rs *RequestStatus) Timeout() RequestStatus {
	return ReqTimeout
}
