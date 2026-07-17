package request

import (
	"context"
	"errors"
	"gateway/internal/common"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"gateway/internal/log"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGetHandler(t *testing.T) {

	timeout := 2.0

	rateLimitOpts := config.ServiceRateLimitOpts{
		RateLimitPerMinute:            nil,
		RouteLevelRateLimitsPerMinute: nil,
	}
	validatorOpts := config.ServiceValidatorOpts{
		RequiredHeaders:   nil,
		RestrictedHeaders: nil,
		AllowedHeaders:    nil,
	}
	routingOpts := config.ServiceReqRoutingOpts{ReqRoutingConfigMap: nil}
	proxyOpts := config.ServiceReqProxyOpts{ProxyReqTimeoutInSeconds: &timeout}
	deprecationOpts := config.ServiceDepricationOpts{
		DeprecatedUrls:    nil,
		DeprecatedHeaders: nil,
		ObsoleteUrls:      nil,
	}
	policyOpts := config.ServicePolicyOpts{}
	versionOpts := config.ServiceVersionOpts{
		AvialableVersions: []string{"v1", "v2"},
		DefaultVersion:    "v2",
	}

	configService := config.ServiceConfig{
		ServiceName:      "test-service",
		TimeoutInSeconds: &timeout,
		RateLimitOpts:    &rateLimitOpts,
		RoutingOpts:      &routingOpts,
		ReqProxyOpts:     &proxyOpts,
		AuthOpts: config.ServcieAuthOpts{
			ValidatorOpts:   &validatorOpts,
			PolicyOpts:      &policyOpts,
			VersionOpts:     &versionOpts,
			DeprecationOpts: &deprecationOpts,
		},
	}

	globalTimeout := 10.0
	c := config.Config{
		Port:                   5000,
		GlobalTimeoutInSeconds: globalTimeout,
		RateLimitPerMinute:     10,
		ReqSizeLimitInBytes:    3000,
		Services: []config.ServiceConfig{
			configService,
		},
		BlockedIps:                   []string{},
		HealthCheckIntervalInSeconds: int64(globalTimeout),
	}

	remoteAddr := "0.0.0.0" + ":3000"
	endpoint := "/test-service/v1"
	r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
	r.RemoteAddr = remoteAddr
	defer r.Body.Close()

	responseBodyText := "hello world"

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "request should successfully move through the pipeline and return a response",
			t: func(t *testing.T) {

				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseBodyText)),
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					}}

				w := httptest.NewRecorder()
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				testData := InitializeReqHanlderWithMocks(t, c)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", w, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, r.URL.Path, remoteAddr).Return(nil)
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, &body, r.Header, r.URL, &proxyOpts).Return(resp, nil)

				req := r.WithContext(common.WithLogData(r.Context(), log.NewLogData(r)))
				ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
				defer cancel()
				testData.Handler.ServeHTTP(w, req.WithContext(ctx))

				assert.Equal(t, http.StatusOK, w.Result().StatusCode, "the status codes should be equal")

				data, err := io.ReadAll(w.Result().Body)
				assert.NoError(t, err, "failed while reading response body")

				assert.Equal(t, responseBodyText, string(data))
				assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
			},
		},
		{
			name: "request should move through the pipeline in a specific order and there should be no reordering",
			t: func(t *testing.T) {

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				w := httptest.NewRecorder()

				testData := InitializeReqHanlderWithMocks(t, c)
				callOrder := make([]int, 0)

				addCall := func(i int) func(args mock.Arguments) {
					callOrder = append(callOrder, i)
					return func(args mock.Arguments) {
					}
				}

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				testData.Mc.On("Load").Return(c, nil).Run(addCall(1))
				testData.Mss.On("Map", endpoint).Return(configService).Run(addCall(2))
				testData.Mv.On("Validate", w, mock.Anything, configService.ServiceName).Return(nil).Run(addCall(4))
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, r.URL.Path, remoteAddr).Return(nil).Run(addCall(5))
				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, &body, r.Header, r.URL, &proxyOpts).Return(resp, nil).Run(addCall(6))

				req := r.WithContext(common.WithLogData(r.Context(), log.NewLogData(r)))
				testData.Handler.ServeHTTP(w, req)

				assert.Equal(t, []int{1, 2, 4, 5, 6}, callOrder)

			},
		},
		{
			name: "request should terminate if it encounters a terminal error",
			t: func(t *testing.T) {

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				w := httptest.NewRecorder()

				testData := InitializeReqHanlderWithMocks(t, c)

				testData.Mc.On("Load").Return(c, nil)
				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", w, mock.Anything, configService.ServiceName).Return(
					customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqValidationFailed, errors.New("request validation failed")),
				)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, r.URL.Path, remoteAddr).Return(nil)

				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool { return true }), r.Method, r.Body, r.Header, r.URL, &proxyOpts).Return(resp, nil)

				req := r.WithContext(common.WithLogData(r.Context(), log.NewLogData(r)))
				testData.Handler.ServeHTTP(w, req)

				assert.Equal(t, http.StatusBadRequest, w.Result().StatusCode)

			},
		},
		{
			name: "request should terminate before proxy if the deadline is exceeded ",
			t: func(t *testing.T) {

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}
				w := httptest.NewRecorder()

				testData := InitializeReqHanlderWithMocks(t, c)

				testData.Mc.On("Load").Return(c, nil)
				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", w, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, r.URL.Path, remoteAddr).Return(nil).Run(
					func(args mock.Arguments) {
						time.Sleep(2 * time.Second)
					})

				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool { return true }), r.Method, &body, r.Header, r.URL, &proxyOpts).Return(resp, nil)

				assert.NoError(t, err)

				req := r.WithContext(common.WithLogData(r.Context(), log.NewLogData(r)))
				ctx, cancel := context.WithTimeout(req.Context(), 1*time.Second)
				cancel()
				testData.Handler.ServeHTTP(w, req.WithContext(ctx))

				assert.Equal(t, http.StatusRequestTimeout, w.Result().StatusCode)

			},
		},
		{
			name: "req should terminate proxy if the context times out",
			t: func(t *testing.T) {

				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)

				w := httptest.NewRecorder()

				testData := InitializeReqHanlderWithMocks(t, c)

				testData.Mc.On("Load").Return(c, nil)

				testData.Mss.On("Map", endpoint).Return(configService)
				testData.Mv.On("Validate", w, mock.Anything, configService.ServiceName).Return(nil)
				testData.Mv.On("ValidateReqSize", 0).Return(nil)
				testData.Mrl.On("Limit", configService.ServiceName, r.URL.Path, remoteAddr).Return(nil)

				testData.Mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, &body, r.Header, r.URL, &proxyOpts).Return(http.Response{}, context.DeadlineExceeded)

				req := r.WithContext(common.WithLogData(r.Context(), log.NewLogData(r)))
				ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
				defer cancel()

				testData.Handler.ServeHTTP(w, req.WithContext(ctx))
				assert.Equal(t, http.StatusRequestTimeout, w.Result().StatusCode)

			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}

}
