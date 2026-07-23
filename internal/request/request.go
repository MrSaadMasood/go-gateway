package request

import (
	"context"
	"errors"
	"gateway/internal/common"
	"gateway/internal/config"
	"gateway/internal/enums"
	"gateway/internal/log"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/route"
	"gateway/internal/services"
	"gateway/internal/telemeter"
	"gateway/internal/validate"
	"io"
	"net/http"
	"strings"
	"time"
)

type HandleRequestData struct {
	Config      config.Config
	Validator   validate.Validator
	Proxier     proxy.Proxier
	Router      route.Router
	RateLimiter ratelimit.RateLimiter
	Telemter    telemeter.Recorder
	GetService  services.GetServiceFunc
}

func NewHandler(hrd HandleRequestData) (http.Handler, error) {

	c := hrd.Config
	storer := hrd.GetService(c.Services)

	mapServiceToReqSuccess := func(config.Config) ActionFunc {
		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {

			service, err := storer.Map(r.URL.Path)
			if err != nil {
				return nil, err
			}

			hrd.Telemter.Record(service.ServiceName, r.URL.Path)
			ctx := common.WithMappedService(r.Context(), service)
			req := r.WithContext(ctx)

			ld, err := common.LogDataFrom(r.Context())
			if err != nil {
				return nil, err
			}
			ld.SetServiceStatusData(log.ServiceStatusData{
				ServiceName: service.ServiceName,
				Status:      enums.ServiceAvailable,
			})

			return req, nil
		}
	}

	validateReqSuccess := func(config.Config) ActionFunc {
		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {

			service, err := common.MappedServiceFrom(r.Context())
			if err != nil {
				return nil, err
			}

			err = hrd.Validator.Validate(w, r, service.ServiceName)
			if err != nil {
				return nil, err
			}

			return r, nil

		}

	}

	rateLimitReqSuccess := func(config.Config) ActionFunc {
		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {

			service, err := common.MappedServiceFrom(r.Context())
			if err != nil {
				return nil, err
			}
			err = hrd.RateLimiter.Limit(service.ServiceName, r.URL.Path, r.RemoteAddr)
			if err != nil {
				return nil, err
			}

			return r, nil

		}
	}

	proxyReqSuccess := func(config.Config) ActionFunc {

		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
			service, err := common.MappedServiceFrom(r.Context())
			if err != nil {
				return nil, err
			}

			reqCtx := r.Context()

			deadline, ok := reqCtx.Deadline()
			if !ok {
				return nil, errors.New("no deadline set on the request")
			}

			timeout := time.Until(deadline)
			if timeout <= 0 {
				return nil, context.DeadlineExceeded
			}

			timeoutCtx, cancel := context.WithTimeout(r.Context(), timeout)
			defer cancel()

			body, err := io.ReadAll(r.Body)
			defer r.Body.Close()
			if err != nil {
				return nil, err
			}

			ld, err := common.LogDataFrom(r.Context())
			if err != nil {
				return nil, err
			}
			ld.SetReqPayload(&body)

			err = hrd.Validator.ValidateReqSize(len(body))
			if err != nil {
				return nil, err
			}

			route, err := hrd.Router.Route(r, service.RoutingOpts)
			if err != nil {
				return nil, err
			}

			res, err := hrd.Proxier.Proxy(timeoutCtx, route, r.Method, &body, r.Header, r.URL, service.ReqProxyOpts)
			if err != nil {
				return nil, err
			}

			ctx := WithProxyRes(reqCtx, &res)

			return r.WithContext(ctx), nil

		}
	}

	var sendResponse = func(reqStatus enums.RequestStatus) ActionFunc {
		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
			res, err := ProxyResFrom(r.Context())
			if err != nil {
				return nil, err
			}

			for k, v := range res.Header {
				w.Header().Set(k, strings.Join(v, ","))
			}
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, err
			}

			ld, err := common.LogDataFrom(r.Context())
			if err != nil {
				return nil, err
			}

			t, err := time.Parse(time.RFC3339, ld.CreatedAt)
			if err != nil {
				return nil, err
			}

			ld.SetResponse(log.LogResData{
				ResPayload: string(body),
				ResHeaders: w.Header(),
				ResTime:    time.Until(t).Seconds(),
			})
			ld.SetFinalReqStatus(reqStatus)

			w.WriteHeader(res.StatusCode)
			w.Write(body)

			return r, nil
		}
	}

	failRequestWithStatus := func(s enums.RequestStatus) ActionFunc {
		return func(w http.ResponseWriter, r *http.Request) (*http.Request, error) {
			ctx := common.WithReqStatus(r.Context(), s)
			return r.WithContext(ctx), nil
		}
	}

	keyActionMap := reqTransitionKeyActionMap{
		{event: enums.MapSuccessReqEvent, from: enums.ReqInitialized}:              mapServiceToReqSuccess(c),
		{event: enums.ValidationSuccessReqEvent, from: enums.ReqServiceMapSuccess}: validateReqSuccess(c),
		{event: enums.RateLimitSuccessReqEvent, from: enums.ReqValidationSuccess}:  rateLimitReqSuccess(c),
		{event: enums.ProxySuccessReqEvent, from: enums.ReqRateLimitSuccess}:       proxyReqSuccess(c),
		{event: enums.ReqSuccessEvent, from: enums.ReqProxySuccess}:                sendResponse(enums.ReqSuccess),

		{event: enums.MapFailedReqEvent, from: enums.ReqInitialized}:              failRequestWithStatus(enums.ReqServiceMapFailed),
		{event: enums.ValidationFailedReqEvent, from: enums.ReqServiceMapSuccess}: failRequestWithStatus(enums.ReqValidationFailed),
		{event: enums.RateLimitFailedReqEvent, from: enums.ReqValidationSuccess}:  failRequestWithStatus(enums.ReqRateLimitFailed),
		{event: enums.ProxyFailedReqEvent, from: enums.ReqRateLimitSuccess}:       failRequestWithStatus(enums.ReqProxyFailed),

		{event: enums.ReqFailedEvent, from: enums.ReqServiceMapFailed}: failRequestWithStatus(enums.ReqFailed),
		{event: enums.ReqFailedEvent, from: enums.ReqValidationFailed}: failRequestWithStatus(enums.ReqFailed),
		{event: enums.ReqFailedEvent, from: enums.ReqRateLimitFailed}:  failRequestWithStatus(enums.ReqFailed),
		{event: enums.ReqFailedEvent, from: enums.ReqProxyFailed}:      failRequestWithStatus(enums.ReqFailed),

		{event: enums.ReqTimeoutEvent, from: enums.ReqInitialized}:       failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqServiceMapSuccess}: failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqServiceMapFailed}:  failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqValidationSuccess}: failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqValidationFailed}:  failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqRateLimitSuccess}:  failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqRateLimitFailed}:   failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqProxySuccess}:      failRequestWithStatus(enums.ReqTimeout),
		{event: enums.ReqTimeoutEvent, from: enums.ReqProxyFailed}:       failRequestWithStatus(enums.ReqTimeout),
	}
	machine := NewReqStateMachine(keyActionMap)

	handler := machine.initializeRequest(
		machine.serviceMapM(
			machine.validationM(
				machine.rateLimiterM(
					machine.proxyM(
						machine.sendSuccessM(),
					),
				),
			),
		),
	)

	timeoutMiddleware := func(c config.Config) func(http.Handler) http.Handler {
		return func(h http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				service, err := storer.Map(r.URL.Path)
				if err != nil {
					sendError(err, w, r)
					return
				}
				ctx, cancel := context.WithTimeout(r.Context(), service.GetProxyTimeout(time.Duration(c.GlobalTimeoutInSeconds)))
				defer cancel()
				h.ServeHTTP(w, r.WithContext(ctx))
			})
		}
	}

	middleware := timeoutMiddleware(c)
	th := middleware(handler)

	return th, nil
}
