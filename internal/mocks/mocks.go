package mocks

import (
	"context"
	"gateway/internal/config"
	"net/http"
	"net/url"

	"github.com/stretchr/testify/mock"
)

type MockConfig struct {
	mock.Mock
}

func (mc *MockConfig) Load() (config.Config, error) {
	args := mc.Called()
	return args.Get(0).(config.Config), args.Error(1)
}

type MockValidator struct {
	mock.Mock
}

func (mv *MockValidator) Validate(w http.ResponseWriter, req *http.Request, serviceName string) error {
	args := mv.Called(w, req, serviceName)
	return args.Error(0)
}

func (mv *MockValidator) ValidateReqSize(bodySizeInBytes int) error {
	args := mv.Called(bodySizeInBytes)
	return args.Error(0)
}

type MockProxier struct{ mock.Mock }

func (mp *MockProxier) Proxy(ctx context.Context, method string, body *[]byte, h http.Header, url *url.URL, ro *config.ServiceReqProxyOpts) (http.Response, error) {
	args := mp.Called(ctx, method, body, h, url, ro)
	return args.Get(0).(http.Response), args.Error(1)
}

type MockRateLimiter struct{ mock.Mock }

func (mrl *MockRateLimiter) Limit(serviceName, path, ip string) error {
	args := mrl.Called(serviceName, path, ip)
	return args.Error(0)
}

type MockServiceStore struct{ mock.Mock }

func (mss *MockServiceStore) Map(path string) (config.ServiceConfig, error) {
	args := mss.Called(path)
	return args.Get(0).(config.ServiceConfig), nil
}

type MockTelemeter struct{}

func (mt *MockTelemeter) Record(serviceName, path string) {
}

func (mt *MockTelemeter) GetTelemetery(serviceName, path string) (uint64, uint64, error) {
	return 1, 1, nil
}

type MockStore struct{ mock.Mock }

func (ms *MockStore) Initialize() error {
	args := ms.Called()
	return args.Error(0)
}

type MockConfigLoader struct {
	mock.Mock
}

func (mcl *MockConfigLoader) Load() (config.Config, error) {
	args := mcl.Called()
	return args.Get(0).(config.Config), args.Error(1)
}
