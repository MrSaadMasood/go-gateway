package auditor

import (
	"context"
	"gateway/internal/config"
	"gateway/internal/log"
	"gateway/internal/request"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

	configService := config.ServiceConfig{
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
		configService.ServiceName: configService,
	}

	c := config.Config{
		Port:                   5000,
		GlobalTimeoutInSeconds: 10,
		RateLimitPerMinute:     10,
		ReqSizeLimitInBytes:    3000,
		Services: []config.ServiceConfig{
			configService,
		},
		BlockedIps:                   []string{},
		HealthCheckIntervalInSeconds: 10,
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var recorder telemeter.Recorder = telemeter.NewReqTelemeter(scm)
	var storer store.Storer = mockStore{}
	reqAuditor := NewReqAuditor(ctx, storer)
	endpoint := "/test-service/v1"

	responseBodyText := "hello world"

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "should record the request properly",
			t: func(t *testing.T) {

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}

				testData := request.InitializeReqHanlderWithMocks(t, c)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", mock.Anything, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(resp, nil)

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				w := httptest.NewRecorder()

				handler := NewHandler(reqAuditor, recorder, testData.Handler)
				handler.ServeHTTP(w, r)

				r1 := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				w1 := httptest.NewRecorder()
				handler.ServeHTTP(w1, r1)

				r3 := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				w3 := httptest.NewRecorder()
				handler.ServeHTTP(w3, r3)

				assert.Greater(t, reqAuditor.processedLogCount(), 0)

				reqAuditor.log()

				assert.Equal(t, reqAuditor.processedLogCount(), 0)
			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}
}
