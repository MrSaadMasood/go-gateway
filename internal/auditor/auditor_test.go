package auditor

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/log"
	"gateway/internal/request"
	"gateway/internal/route"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type mockStore struct {
	mock.Mock
}

func (ms *mockStore) StoreLogs(l []log.LogData) error {
	args := ms.Called(l)
	return args.Error(0)
}

func (ms *mockStore) ReadLogsHandler() http.Handler {
	args := ms.Called()
	return args.Get(0).(http.Handler)

}

type mockLogger struct {
	mock.Mock
}

func (ml *mockLogger) Log(level slog.Level, msg string, args ...any) {
	ml.Called(level, msg, args)
}

func TestAuditHandler(t *testing.T) {

	configService := config.ServiceConfig{
		ServiceName:      "test-service",
		ServiceUrl:       "https://test-service.com",
		TimeoutInSeconds: nil,
		RateLimitOpts:    nil,
		RoutingOpts:      nil,
		ReqProxyOpts:     nil,
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

	endpoint := "/test-service/v1"
	gatewayEndpoint := "https://gateway" + endpoint

	responseBodyText := "hello world"

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "should record the request properly",
			t: func(t *testing.T) {

				mStore := new(mockStore)
				mLogger := new(mockLogger)
				mStore.On("StoreLogs", mock.Anything).Return(nil)
				mLogger.On("Log", mock.Anything, mock.AnythingOfType("string"), mock.Anything)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var recorder telemeter.Recorder = telemeter.NewReqTelemeter(scm)
				var logger log.Logger = mLogger
				var storer store.Storer = mStore

				reqAuditor := NewReqAuditor(ctx, storer, logger)

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				r := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				route, err := route.NewReqRouter().Route(r, nil)
				require.NoError(t, err)

				testData := request.InitializeReqHanlderWithMocks(t, c)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", mock.Anything, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), configService.ServiceUrl+route, mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Return(resp, nil)

				w := httptest.NewRecorder()

				handler := NewHandler(reqAuditor, recorder, testData.Handler)
				handler.ServeHTTP(w, r)

				r1 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w1 := httptest.NewRecorder()
				handler.ServeHTTP(w1, r1)

				r3 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w3 := httptest.NewRecorder()
				handler.ServeHTTP(w3, r3)

				assert.Greater(t, reqAuditor.processedLogCount(), 0)

				reqAuditor.log()

				assert.Equal(t, reqAuditor.processedLogCount(), 0)
			},
		},
		{
			name: "should fallback to the logger if storing the logs fails",
			t: func(t *testing.T) {

				mStore := new(mockStore)
				mLogger := new(mockLogger)

				mStore.On("StoreLogs", mock.Anything).Return(errors.New("unexpected error"))
				mLogger.On("Log", mock.Anything, mock.AnythingOfType("string"), mock.Anything)

				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()
				var recorder telemeter.Recorder = telemeter.NewReqTelemeter(scm)
				var logger log.Logger = mLogger
				var storer store.Storer = mStore

				reqAuditor := NewReqAuditor(ctx, storer, logger)

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				r := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				route, err := route.NewReqRouter().Route(r, nil)
				require.NoError(t, err)

				testData := request.InitializeReqHanlderWithMocks(t, c)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", mock.Anything, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), configService.ServiceUrl+route, mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Return(resp, nil)

				w := httptest.NewRecorder()

				handler := NewHandler(reqAuditor, recorder, testData.Handler)
				handler.ServeHTTP(w, r)

				r1 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w1 := httptest.NewRecorder()
				handler.ServeHTTP(w1, r1)

				r3 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w3 := httptest.NewRecorder()
				handler.ServeHTTP(w3, r3)

				assert.Greater(t, reqAuditor.processedLogCount(), 0)

				reqAuditor.log()
				mLogger.AssertExpectations(t)

			},
		},
		{
			name: "should dump to logger if the server shutsdown",
			t: func(t *testing.T) {

				mStore := new(mockStore)
				mLogger := new(mockLogger)

				mStore.On("StoreLogs", mock.Anything).Return(errors.New("unexpected error"))
				mLogger.On("Log", mock.Anything, mock.AnythingOfType("string"), mock.Anything)

				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				var recorder telemeter.Recorder = telemeter.NewReqTelemeter(scm)
				var logger log.Logger = mLogger
				var storer store.Storer = mStore

				reqAuditor := NewReqAuditor(ctx, storer, logger)

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				r := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				route, err := route.NewReqRouter().Route(r, nil)
				require.NoError(t, err)

				testData := request.InitializeReqHanlderWithMocks(t, c)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", mock.Anything, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, mock.AnythingOfType("string"), mock.AnythingOfType("string")).Return(nil)
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), configService.ServiceUrl+route, mock.AnythingOfType("string"), mock.Anything, mock.Anything, mock.Anything).Return(resp, nil)

				w := httptest.NewRecorder()

				handler := NewHandler(reqAuditor, recorder, testData.Handler)
				handler.ServeHTTP(w, r)

				r1 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w1 := httptest.NewRecorder()
				handler.ServeHTTP(w1, r1)

				r3 := httptest.NewRequest(http.MethodGet, gatewayEndpoint, http.NoBody)
				w3 := httptest.NewRecorder()
				handler.ServeHTTP(w3, r3)

				assert.Greater(t, reqAuditor.processedLogCount(), 0)

				reqAuditor.flushLogsPeriodically()
				mLogger.AssertExpectations(t)

			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}
}
