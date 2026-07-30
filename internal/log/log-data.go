package log

import (
	"gateway/internal/enums"
	logger "log"
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
	ReqStatus  []enums.RequestStatus
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
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
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
				ReqStatus:  make([]enums.RequestStatus, 0),
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
		logger.Println("Failed to set req payload for auditor: ", string(*p), "the log data: ", ld.Data)
		return
	}

	ld.Data.LogReqData.ReqPayload = string(*p)
}

func (ld *LogData) AppendRequestStatus(s enums.RequestStatus) {
	if ld.Data.LogReqData == nil {
		logger.Println("Failed to set req payload for auditor the log data: ", ld.Data)
		return
	}

	ld.Data.LogReqData.ReqStatus = append(ld.Data.LogReqData.ReqStatus, s)

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
