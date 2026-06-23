package log

import (
	"gateway/internal/enums"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type FailureData struct {
	Reason string
}

type LogReqData struct {
	ReqId      string
	ReqMethod  string
	ReqPayload string
	ReqHeaders http.Header
	ReqPath    string
	ReqStatus  enums.RequestStatus
}

type LogResData struct {
	ResPayload string
	ResHeaders http.Header
	ResTime    float64
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
	*LogResData
	*ServiceStatusData
	*LogReqData
	*ServiceTrafficData
}

type LogData struct {
	Source    string
	CreatedAt string
	Data      LogPayload
}

func NewLogData(r *http.Request) *LogData {

	reqId := uuid.New().String()

	logData := &LogData{
		Source:    "Auditor",
		CreatedAt: time.Now().UTC().String(),
		Data: LogPayload{
			FailureData:        nil,
			LogResData:         nil,
			ServiceStatusData:  nil,
			ServiceTrafficData: nil,
			LogReqData: &LogReqData{
				ReqId:      reqId,
				ReqMethod:  r.Method,
				ReqHeaders: r.Header,
				ReqPath:    r.URL.Path,
			},
		},
	}
	return logData
}

func (ld *LogData) SetServiceTrafficData(sstd ServiceTrafficData) {
	ld.Data.ServiceTrafficData = &sstd
}

func (ld *LogData) SetReqPayload(p *[]byte) {

	if ld.Data.LogReqData == nil || p == nil {
		return
	}

	ld.Data.LogReqData.ReqPayload = string(*p)
}

func (ld *LogData) SetFinalReqStatus(s enums.RequestStatus) {
	if ld.Data.LogReqData == nil {
		return
	}
	ld.Data.LogReqData.ReqStatus = s
	if ld.Data.ServiceTrafficData == nil {
		return
	}

}

func (ld *LogData) SetResponse(rs LogResData) {
	ld.Data.LogResData = &rs
}

func (ld *LogData) SetFailure(fd FailureData) {
	ld.Data.FailureData = &fd
}

func (ld *LogData) SetServiceStatusData(ssd ServiceStatusData) {
	ld.Data.ServiceStatusData = &ssd
}
