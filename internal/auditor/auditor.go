package auditor

import (
	"context"
	"gateway/internal/common"
	"gateway/internal/log"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	logger "log"
	"net/http"
	"time"
)

type Auditor interface {
	Log(log.LogData)
}

type auditData struct {
	processed bool
	logData   *log.LogData
}

type reqAuditor struct {
	ctx             context.Context
	reqAuditDataMap map[string]*auditData
	storage         store.Storer
	flushDuration   time.Duration
}

func NewReqAuditor(ctx context.Context, storage store.Storer) *reqAuditor {
	ra := reqAuditor{ctx, make(map[string]*auditData), storage, 5 * time.Minute}
	return &ra
}

func (ra *reqAuditor) log() {
	logs := make([]log.LogData, 0)
	for reqId, auditData := range ra.reqAuditDataMap {
		if auditData.processed {
			logs = append(logs, *auditData.logData)
			delete(ra.reqAuditDataMap, reqId)
		}
	}
	err := ra.storage.StoreLogs(logs)
	if err != nil {
		logger.Println("Error occured while storing the request logs:", err.Error())
	}
}

func (ra *reqAuditor) processedLogCount() int {
	count := 0
	for _, ad := range ra.reqAuditDataMap {
		if ad.processed == true {
			count++
		}
	}
	return count
}

func (ra *reqAuditor) flushLogsPeriodically() {

	t := time.NewTicker(ra.flushDuration)
	for {
		select {
		case <-t.C:
			ra.log()
		case <-ra.ctx.Done():
			t.Stop()
			return
		}
	}

}

func NewHandler(ra *reqAuditor, recorder telemeter.Recorder, h http.Handler) http.Handler {
	go ra.flushLogsPeriodically()

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		ld := log.NewLogData(r)
		ad := &auditData{
			processed: false,
			logData:   ld,
		}

		ra.reqAuditDataMap[ld.Data.ReqId] = ad
		ctx := common.WithLogData(r.Context(), ld)

		h.ServeHTTP(w, r.WithContext(ctx))

		ad.processed = true

		service, err := common.MappedServiceFrom(r.Context())
		if err != nil {
			return
		}

		serviceTraffic, routeTraffic, err := recorder.GetTelemetery(service.ServiceName, r.URL.Path)
		if err != nil {
			return
		}

		ld.SetServiceTrafficData(log.ServiceTrafficData{
			ServiceName: service.ServiceName,
			Traffic:     serviceTraffic,
			ServicePathReqTrafficData: &log.ServicePathReqTrafficData{
				ReqPath: r.URL.Path,
				Traffic: routeTraffic,
			},
		})

	})
}
