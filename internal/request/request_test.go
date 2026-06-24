package request

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/controller"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"gateway/internal/mocks"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/services"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
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

	sampleTimeout := func(i int) *time.Duration {
		timeout := time.Duration(i) * time.Second
		return &timeout
	}

	rateLimitOpts := config.ServiceRateLimitOpts{
		RateLimit:            nil,
		RouteLevelRateLimits: nil,
	}
	validatorOpts := config.ServiceValidatorOpts{
		RequiredHeaders:   nil,
		RestrictedHeaders: nil,
		AllowedHeaders:    nil,
	}
	redirectOpts := config.ServiceRedirectOpts{RouteLevelRedirection: nil, ProxyReqTimeout: sampleTimeout(2)}
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
		ServiceName:   "test-service",
		ServiceUrl:    "/test-service",
		Timeout:       sampleTimeout(4),
		RateLimitOpts: &rateLimitOpts,
		RedirectOpts:  &redirectOpts,
		AuthOpts: config.ServcieAuthOpts{
			ValidatorOpts:   &validatorOpts,
			PolicyOpts:      &policyOpts,
			VersionOpts:     &versionOpts,
			DeprecationOpts: &deprecationOpts,
		},
	}

	c := config.Config{
		Port:                5000,
		Timeout:             10 * time.Second,
		RateLimit:           10,
		ReqSizeLimitInBytes: 3000,
		Services: []config.ServiceConfig{
			configService,
		},
		InternalOnlyServices:     []string{"test-service-restricted"},
		InternalOnlyServicesUrls: []string{"/test-service-restricted"},
		BlockedIps:               []string{},
		HealthCheckInterval:      10 * time.Second,
	}

	reqRateLimiterOpts := ratelimit.NewReqRateLimiter(context.Background())

	reqAccessControllerOpts := func(path string) controller.AccessControllerOpts {
		return controller.AccessControllerOpts{
			RestrictedServices: c.InternalOnlyServices,
			RestrictedPaths:    c.InternalOnlyServicesUrls,
			ServiceName:        configService.ServiceName,
			ReqPath:            path,
		}
	}
	ip := "0.0.0.0"

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "request should successfully move through the pipeline and return a response",
			t: func(t *testing.T) {

				mc := new(mocks.MockConfig)
				mv := new(mocks.MockValidator)
				mp := new(mocks.MockProxier)
				mrl := new(mocks.MockRateLimiter)
				msac := new(mocks.MockServiceAccessController)
				mss := new(mocks.MockServiceStore)
				mt := new(mocks.MockTelemeter)

				var configLoader config.Loader = mc
				var validtor validate.Validator = mv
				var proxier proxy.Proxier = mp
				var rateLimiter ratelimit.RateLimiter = mrl
				var serviceAccessController controller.ServiceAccessController = msac
				var mockServcieStorer services.Storer = mss
				var mockRecorder telemeter.Recorder = mt
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return mockServcieStorer
				}

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				responseBodyText := "hello world"
				resp := http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(responseBodyText)),
					Header: http.Header{
						"Content-Type": []string{"text/plain"},
					}}
				w := httptest.NewRecorder()
				body, err := io.ReadAll(r.Body)
				assert.NoError(t, err)
				defer w.Result().Body.Close()

				mc.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				msac.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				mv.On("Validate", w, r, configService.ServiceName).Return(nil)
				mrl.On("Limit", configService.ServiceName, r.URL.Path, ip).Return(nil)
				mp.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, &body, r.Header, r.URL, redirectOpts).Return(resp, nil)

				c, err := configLoader.Load()
				assert.NoError(t, err)

				hrd := HandleRequestData{
					Config:                  c,
					Validator:               validtor,
					Proxier:                 proxier,
					RateLimiter:             rateLimiter,
					ServiceAccessController: serviceAccessController,
					GetService:              getServiceFunc,
					Telemter:                mockRecorder,
				}

				handler, err := NewHandler(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				handler.ServeHTTP(w, r)

				assert.Equal(t, http.StatusOK, w.Result().StatusCode, "the status codes should be equal")
				data, err := io.ReadAll(w.Result().Body)
				if err != nil {
					t.Fatal("failed while reading response body", err)
				}
				assert.Equal(t, responseBodyText, string(data))
				assert.Equal(t, "text/plain", w.Header().Get("Content-Type"))
			},
		},
		{
			name: "request should move through the pipeline in a specific order and there should be no reordering",
			t: func(t *testing.T) {

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				responseBodyText := "hello world"
				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}

				configLoader := mocks.MockConfig{}
				validtor := mocks.MockValidator{}
				proxier := mocks.MockProxier{}
				rateLimiter := mocks.MockRateLimiter{}
				serviceAccessController := mocks.MockServiceAccessController{}
				mss := mocks.MockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				callOrder := make([]int, 0)

				addCall := func(i int) func(args mock.Arguments) {
					callOrder = append(callOrder, i)
					return func(args mock.Arguments) {
					}
				}

				configLoader.On("Load").Return(c, nil).Run(addCall(1))
				mss.On("Map", endpoint).Return(configService).Run(addCall(2))
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil).Run(addCall(3))
				validtor.On("Validate", reqValidationOpts).Return(nil).Run(addCall(4))
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil).Run(addCall(5))

				proxier.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, r.Body, r.Header, r.URL, redirectOpts).Return(resp, nil).Run(addCall(6))

				c, err := configLoader.Load()
				assert.NoError(t, err)

				hrd := HandleRequestData{
					Config:                  c,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := NewHandler(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				defer w.Result().Body.Close()

				assert.Equal(t, []int{1, 2, 3, 4, 5, 6}, callOrder)

			},
		},
		{
			name: "request should terminate if it encounters a terminal error",
			t: func(t *testing.T) {

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				responseBodyText := "hello world"
				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}

				configLoader := mocks.MockConfig{}
				validtor := mocks.MockValidator{}
				proxier := mocks.MockProxier{}
				rateLimiter := mocks.MockRateLimiter{}
				serviceAccessController := mocks.MockServiceAccessController{}
				mss := mocks.MockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(
					customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqValidationFailed, errors.New("request validation failed")),
				)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil)

				proxier.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool { return true }), r.Method, r.Body, r.Header, r.URL, redirectOpts).Return(resp, nil)

				c, err := configLoader.Load()
				assert.NoError(t, err)

				hrd := HandleRequestData{
					Config:                  c,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := NewHandler(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				defer w.Result().Body.Close()

			},
		},
		{
			name: "request should terminate before proxy if the deadline is exceeded ",
			t: func(t *testing.T) {

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				responseBodyText := "hello world"
				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}

				configLoader := mocks.MockConfig{}
				validtor := mocks.MockValidator{}
				proxier := mocks.MockProxier{}
				rateLimiter := mocks.MockRateLimiter{}
				serviceAccessController := mocks.MockServiceAccessController{}
				mss := mocks.MockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(nil)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil).Run(func(args mock.Arguments) {
					time.Sleep(3 * time.Second)
				})

				proxier.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool { return true }), r.Method, r.Body, r.Header, r.URL, redirectOpts).Return(resp, nil)

				c, err := configLoader.Load()
				assert.NoError(t, err)

				hrd := HandleRequestData{
					Config:                  c,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := NewHandler(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				defer w.Result().Body.Close()
				rateLimiter.AssertExpectations(t)
				assert.Equal(t, http.StatusRequestTimeout, w.Result().StatusCode)

			},
		},
		{
			name: "req should terminate proxy if the context times out",
			t: func(t *testing.T) {

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()

				configLoader := mocks.MockConfig{}
				validtor := mocks.MockValidator{}
				proxier := mocks.MockProxier{}
				rateLimiter := mocks.MockRateLimiter{}
				serviceAccessController := mocks.MockServiceAccessController{}
				mss := mocks.MockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(nil)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil)

				proxier.On("Proxy", mock.MatchedBy(func(ctx context.Context) bool {
					return true
				}), r.Method, r.Body, r.Header, r.URL, redirectOpts).Return(http.Response{}, context.DeadlineExceeded)
				c, err := configLoader.Load()
				assert.NoError(t, err)

				hrd := HandleRequestData{
					Config:                  c,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := NewHandler(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				defer w.Result().Body.Close()
				assert.Equal(t, http.StatusRequestTimeout, w.Result().StatusCode)

			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}

}
