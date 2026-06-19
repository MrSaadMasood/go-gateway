package auditor

import (
	"context"
	"gateway/internal/common"
	"gateway/internal/enums"
	"net/http"
	"time"

	"github.com/google/uuid"
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

type reqAuditor struct {
	ctx           context.Context
	reqLogDataMap map[string]*LogData
}

func NewHandler(ctx context.Context, h http.Handler) http.Handler {
	ra := reqAuditor{ctx, make(map[string]*LogData)}
	logChan := make(chan *LogData, 5000)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		reqId := uuid.New().String()
		ra.reqLogDataMap[reqId] = &LogData{
			Source:    "Auditor",
			CreatedAt: time.Now(),
			Data:      LogPayload{},
		}

		h.ServeHTTP(w, r)

		logData, ok := ra.reqLogDataMap[reqId]
		if ok {
			sc, err := common.MappedServiceFrom(r.Context())
			mappedService := ""
			if err != nil {
				mappedService = sc.ServiceName
			}

			status, _ := common.StatusFrom(r.Context())

			LogReqData{
				ReqId:                    reqId,
				ReqMethod:                r.Method,
				ReqPayload:               r.Body,
				ReqHeaders:               r.Header,
				ReqPath:                  r.URL.Path,
				TargetBackendServiceName: mappedService,
				ReqStatus:                status,
			}
		}
	})
}

func (a reqAuditor) Log(w http.ResponseWriter, r *http.Request, buffer chan<- *LogData) {
	var ld *LogData
	buffer <- ld
}

func (a reqAuditor) Flush() {

}
