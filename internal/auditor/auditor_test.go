package auditor

import (
	"context"
	"gateway/internal/config"
	"gateway/internal/log"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"net/http"
	"testing"
)

type mockStore struct{}

func (ms mockStore) StoreLogs([]log.LogData) error {
	return nil
}

func (ms mockStore) ReadLogsHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("logs read"))
	})
}

func TestAuditHandler(t *testing.T) {

	testService1 := config.ServiceConfig{
		ServiceName:      "test-service",
		ServiceUrl:       "/test-service",
		TimeoutInSeconds: nil,
		RateLimitOpts:    nil,
		RedirectOpts:     nil,
		AuthOpts: config.ServcieAuthOpts{
			ValidatorOpts:   nil,
			DeprecationOpts: nil,
			PolicyOpts:      nil,
			VersionOpts:     nil,
		},
	}
	scm := config.ServiceConfigMap{
		testService1.ServiceName: testService1,
	}

	ctx, cancel := context.WithCancel(context.Background())

	var recorder telemeter.Recorder = telemeter.NewReqTelemeter(scm)
	var storer store.Storer = mockStore{}
	handler := NewHandler(ctx, recorder, storer)
}
