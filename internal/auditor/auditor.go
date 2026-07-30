package auditor

import (
	"context"
	"gateway/internal/common"
	"gateway/internal/log"
	glog "gateway/internal/log"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"log/slog"
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
	logger          glog.Logger
}

func NewReqAuditor(ctx context.Context, storage store.Storer, logger glog.Logger) *reqAuditor {
	ra := reqAuditor{ctx, make(map[string]*auditData), storage, 5 * time.Minute, logger}
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
		ra.logger.Log(slog.LevelError, "AUDITOR", slog.String("error", err.Error()))
		ra.logToStdout(&logs)
	}
}

func (ra *reqAuditor) logToStdout(logs *[]log.LogData) {
	for _, l := range *logs {
		ra.logger.Log(slog.LevelInfo, "AUDITOR", slog.Any("req_log", l))
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
			logs := make([]log.LogData, len(ra.reqAuditDataMap))
			for _, auditData := range ra.reqAuditDataMap {
				logs = append(logs, *auditData.logData)
			}
			ra.logToStdout(&logs)
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
