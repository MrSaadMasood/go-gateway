package e2e

import (
	"context"
	"gateway/internal/auditor"
	"gateway/internal/config"
	"gateway/internal/gateway"
	glog "gateway/internal/log"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/request"
	"gateway/internal/route"
	"gateway/internal/services"
	"gateway/internal/store"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gavv/httpexpect/v2"
	"github.com/joho/godotenv"
	"github.com/rs/cors"
	"github.com/stretchr/testify/assert"
)

func TestGateway(t *testing.T) {

	err := godotenv.Load("../../.env")
	if err != nil {
		panic(err)
	}
	ctx := context.Background()
	var configLoader config.Loader = config.NewConfigLoader("../../config.json")
	storage := store.NewStorage(ctx)
	var storer store.Storer = storage
	var logger glog.Logger = glog.New(ctx)

	g := gateway.New(ctx, configLoader, storage, logger)
	c := g.Start()
	scm := c.GetServiceConfigMap()

	var validator validate.Validator = validate.NewReqValidator(c.BlockedIps, c.AllowedOrigins, c.ReqSizeLimitInBytes, scm)
	var proxier proxy.Proxier = proxy.NewReqProxy(c.GetGlobalTimeout())
	var rateLimiter ratelimit.RateLimiter = ratelimit.NewReqRateLimiter(ctx, c.RateLimitPerMinute, scm)
	var requestTelemeter telemeter.Recorder = telemeter.NewReqTelemeter(scm)
	var router route.Router = route.NewReqRouter()
	corsPolicy := cors.New(cors.Options{
		AllowedOrigins: append([]string{}, c.AllowedOrigins...),
	})

	reqAuditor := auditor.NewReqAuditor(ctx, storer, logger)

	handler, err := request.NewHandler(request.HandleRequestData{
		Config:      c,
		Validator:   validator,
		RateLimiter: rateLimiter,
		Proxier:     proxier,
		Telemter:    requestTelemeter,
		Router:      router,
		GetService: func(scs []config.ServiceConfig) services.Storer {
			return services.NewServiceStore(scs)
		},
	})

	assert.NoError(t, err)

	handler = corsPolicy.Handler(handler)
	handler = auditor.NewHandler(reqAuditor, requestTelemeter, handler)

	mux := http.NewServeMux()
	mux.Handle("/", handler)
	mux.Handle("GET /logs", storer.ReadLogsHandler())

	server := httptest.NewServer(mux)
	defer server.Close()

	e := httpexpect.Default(t, server.URL)

	e.GET("/json-placeholder/v1/comments?postId=1").WithHeader("Authorization", "Bearer token").Expect().Status(http.StatusOK).JSON().IsArray()

}
