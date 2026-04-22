package auditor

import (
	"gateway/internal/enums"
	"time"
)

type FailureData struct {
	Reason string
}

type LogReqData struct {
	ReqId                    string
	ReqMethod                string
	ReqPayload               any
	ReqHeaders               map[string]string
	ReqPath                  string
	TargetBackendServiceName string
	ReqStatus                enums.RequestStatus
}

type LogResData struct {
	ResPayload any
	ResHeaders map[string]string
	ResTime    time.Time
}
type ServicePathReqTrafficData struct {
	ReqPath string
	Traffic uint64
}

type ServiceTrafficData struct {
	ServiceName string
	Traffic     uint64
	*ServicePathReqTrafficData
}

type ServiceStatusData struct {
	ServiceName string
	Status      enums.ServiceStatus
}

type LogPayload struct {
	*FailureData
	*LogReqData
	*LogResData
	*ServiceStatusData
	*ServiceTrafficData
}

type LogData struct {
	Source    string
	CreatedAt time.Time
	Data      LogPayload
}

type Auditor interface {
	Log(LogData)
}
