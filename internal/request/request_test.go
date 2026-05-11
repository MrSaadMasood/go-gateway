package request

import (
	"errors"
	"gateway/internal/config"
	"gateway/internal/controller"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/services"
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

type mockConfig struct {
	mock.Mock
}

func (mc *mockConfig) Load() (config.Config, error) {
	args := mc.Called()
	return args.Get(0).(config.Config), nil
}

type mockValidator struct {
	mock.Mock
}

func (mv *mockValidator) Validate(opts validate.ValidationOpts) error {
	args := mv.Called(opts)
	return args.Error(0)
}

type mockProxier struct{ mock.Mock }

func (mp *mockProxier) Proxy(r *http.Request, ro config.ServiceRedirectOpts) (http.Response, error) {
	args := mp.Called(r, ro)
	return args.Get(0).(http.Response), nil
}

type mockRateLimiter struct{ mock.Mock }

func (mrl *mockRateLimiter) Limit(rlo ratelimit.RateLimitOpts) error {
	args := mrl.Called(rlo)
	return args.Error(0)
}

type mockServiceAccessController struct{ mock.Mock }

func (msc *mockServiceAccessController) Control(aco controller.AccessControllerOpts) error {
	args := msc.Called(aco)
	return args.Error(0)
}

type mockServiceStore struct{ mock.Mock }

func (mss *mockServiceStore) Map(path string) (config.ServiceConfig, error) {
	args := mss.Called(path)
	return args.Get(0).(config.ServiceConfig), nil
}

func TestGetHandlerFunc(t *testing.T) {

	rateLimitOpts := config.ServiceRateLimitOpts{
		RateLimit:            nil,
		RouteLevelRateLimits: nil,
	}
	validatorOpts := config.ServiceValidatorOpts{
		RequiredBodyFields: nil,
		RequiredHeaders:    nil,
		RestrictedHeaders:  nil,
		AllowedHeaders:     nil,
	}
	redirectOpts := config.ServiceRedirectOpts{
		RouteLevelRedirection: nil,
	}
	urlDeprecationOpts := config.ServiceDepricationOpts{
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
		ServiceName:        "test-service",
		ServiceUrl:         "/test-service",
		Timeout:            nil,
		RateLimitOpts:      &rateLimitOpts,
		ValidatorOpts:      &validatorOpts,
		RedirectOpts:       &redirectOpts,
		UrlDepricationOpts: &urlDeprecationOpts,
		PolicyOpts:         &policyOpts,
		VersionOpts:        &versionOpts,
	}

	c := config.Config{
		Port:         5000,
		Timeout:      10 * time.Second,
		RateLimit:    10,
		ReqSizeLimit: 3000,
		Services: []config.ServiceConfig{
			configService,
		},
		InternalOnlyServices:     []string{"test-service-restricted"},
		InternalOnlyServicesUrls: []string{"/test-service-restricted"},
		BlockedIps:               []string{},
		HealthCheckInterval:      10 * time.Second,
	}

	reqValidationOpts := validate.ValidationOpts{
		GlobalBlockedIps:   c.BlockedIps,
		GlobalReqSizeLimit: c.ReqSizeLimit,
		ServiceOpts:        configService,
	}

	reqRateLimiterOpts := ratelimit.RateLimitOpts{
		GlobalRouteLimits:    c.RateLimit,
		ServiceRateLimitOpts: rateLimitOpts,
	}

	reqAccessControllerOpts := func(path string) controller.AccessControllerOpts {
		return controller.AccessControllerOpts{
			RestrictedServices: c.InternalOnlyServices,
			RestrictedPaths:    c.InternalOnlyServicesUrls,
			ServiceName:        configService.ServiceName,
			ReqPath:            path,
		}
	}

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "request should successfully move through the pipeline and return a response",
			t: func(t *testing.T) {

				configLoader := mockConfig{}
				validtor := mockValidator{}
				proxier := mockProxier{}
				rateLimiter := mockRateLimiter{}
				serviceAccessController := mockServiceAccessController{}
				mss := mockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
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

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(nil)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil)
				proxier.On("Proxy", r, redirectOpts).Return(resp, nil)

				hrd := HandleRequestData{
					ConfigLoader:            &configLoader,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := GetHandlerFunc(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler(w, r)
				defer w.Result().Body.Close()

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

				configLoader := mockConfig{}
				validtor := mockValidator{}
				proxier := mockProxier{}
				rateLimiter := mockRateLimiter{}
				serviceAccessController := mockServiceAccessController{}
				mss := mockServiceStore{}
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
				proxier.On("Proxy", r, redirectOpts).Return(resp, nil).Run(addCall(6))

				hrd := HandleRequestData{
					ConfigLoader:            &configLoader,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := GetHandlerFunc(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler(w, r)
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

				configLoader := mockConfig{}
				validtor := mockValidator{}
				proxier := mockProxier{}
				rateLimiter := mockRateLimiter{}
				serviceAccessController := mockServiceAccessController{}
				mss := mockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(
					customerrors.ReqFailedErr{Code: http.StatusBadRequest, Message: "request validation failed", Status: enums.ReqValidationFailed},
				)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil)
				proxier.On("Proxy", r, redirectOpts).Return(resp, nil)

				hrd := HandleRequestData{
					ConfigLoader:            &configLoader,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := GetHandlerFunc(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler(w, r)
				defer w.Result().Body.Close()
				assert.Equal(t, w.Result().StatusCode, http.StatusBadRequest)

			},
		},
		{
			name: "request should continue to the next stage if the error is no terminal",
			t: func(t *testing.T) {

				endpoint := "/test-service/v1"
				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				responseBodyText := "hello world"
				resp := http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(responseBodyText))}

				configLoader := mockConfig{}
				validtor := mockValidator{}
				proxier := mockProxier{}
				rateLimiter := mockRateLimiter{}
				serviceAccessController := mockServiceAccessController{}
				mss := mockServiceStore{}
				var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
					return &mss
				}

				configLoader.On("Load").Return(c, nil)
				mss.On("Map", endpoint).Return(configService)
				serviceAccessController.On("Control", reqAccessControllerOpts(r.URL.Path)).Return(nil)
				validtor.On("Validate", reqValidationOpts).Return(
					errors.Errorf("non terminal error"),
				)
				rateLimiter.On("Limit", reqRateLimiterOpts).Return(nil)
				proxier.On("Proxy", r, redirectOpts).Return(resp, nil)

				hrd := HandleRequestData{
					ConfigLoader:            &configLoader,
					Validator:               &validtor,
					Proxier:                 &proxier,
					RateLimiter:             &rateLimiter,
					ServiceAccessController: &serviceAccessController,
					GetService:              getServiceFunc,
				}

				handler, err := GetHandlerFunc(hrd)
				if err != nil {
					t.Fatal("failed to get handler", err)
				}

				w := httptest.NewRecorder()
				handler(w, r)
				defer w.Result().Body.Close()
				assert.Equal(t, w.Result().StatusCode, http.StatusOK)

			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}

}

func TestNewRequest(t *testing.T) {

	testTable := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "it should successfully initialize a new request",
			t: func(t *testing.T) {
				r := NewRequest()
				assert.Equal(t, enums.ReqInitialized, r.state)
			},
		},
		{
			name: "it should successfully transition the request to the next state",
			t: func(t *testing.T) {
				r := NewRequest()
				err := r.NextState(enums.ReqServiceMapSuccess)
				assert.Equal(t, nil, err)

			},
		},
		{
			name: "it should not allow random state jumps for the request",
			t: func(t *testing.T) {
				r := NewRequest()
				err := r.NextState(enums.ReqSuccess)
				assert.Error(t, err)
			},
		},
		{
			name: "it should return a terminal error if the terminal state is reached",
			t: func(t *testing.T) {
				r := NewRequest()
				err := r.NextState(enums.ReqServiceMapFailed)
				assert.NoError(t, err)
				result := r.hasReachedTerminalState()
				assert.Equal(t, true, result)
			},
		},
	}

	for _, tt := range testTable {
		t.Run(tt.name, tt.t)

	}
}
