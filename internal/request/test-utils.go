package request

import (
	"gateway/internal/config"
	"gateway/internal/mocks"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/services"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func InitializeReqHanlderWithMocks(t *testing.T, cfg config.Config) struct {
	Mc      *mocks.MockConfig
	Mv      *mocks.MockValidator
	Mp      *mocks.MockProxier
	Mrl     *mocks.MockRateLimiter
	Mss     *mocks.MockServiceStore
	Mt      *mocks.MockTelemeter
	Handler http.Handler
} {

	t.Helper()

	mc := new(mocks.MockConfig)
	mv := new(mocks.MockValidator)
	mp := new(mocks.MockProxier)
	mrl := new(mocks.MockRateLimiter)
	mss := new(mocks.MockServiceStore)
	mt := new(mocks.MockTelemeter)
	mc.On("Load").Return(cfg, nil)

	var configLoader config.Loader = mc
	var validtor validate.Validator = mv
	var proxier proxy.Proxier = mp
	var rateLimiter ratelimit.RateLimiter = mrl
	var mockServcieStorer services.Storer = mss
	var mockRecorder telemeter.Recorder = mt
	var getServiceFunc services.GetServiceFunc = func(scs []config.ServiceConfig) services.Storer {
		return mockServcieStorer
	}

	c, err := configLoader.Load()
	assert.NoError(t, err)

	hrd := HandleRequestData{
		Config:      c,
		Validator:   validtor,
		Proxier:     proxier,
		RateLimiter: rateLimiter,
		GetService:  getServiceFunc,
		Telemter:    mockRecorder,
	}

	handler, err := NewHandler(hrd)
	if err != nil {
		t.Fatal("failed to get handler", err)
	}

	return struct {
		Mc      *mocks.MockConfig
		Mv      *mocks.MockValidator
		Mp      *mocks.MockProxier
		Mrl     *mocks.MockRateLimiter
		Mss     *mocks.MockServiceStore
		Mt      *mocks.MockTelemeter
		Handler http.Handler
	}{
		Mc:      mc,
		Mv:      mv,
		Mp:      mp,
		Mrl:     mrl,
		Mss:     mss,
		Mt:      mt,
		Handler: handler,
	}
}
