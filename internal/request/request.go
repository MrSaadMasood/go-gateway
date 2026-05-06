package request

import (
	"errors"
	"fmt"
	"gateway/internal/config"
	"gateway/internal/controller"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"gateway/internal/proxy"
	ratelimit "gateway/internal/rate-limit"
	"gateway/internal/services"
	"gateway/internal/validate"
	"io"
	"net/http"
	"strings"
)

type request struct {
	state enums.RequestStatus
}

func NewRequest() request {
	return request{
		state: enums.ReqInitialized,
	}
}

func (r *request) NextState(to enums.RequestStatus) error {
	err := r.state.Next(r.state, to)
	if err != nil {
		if errors.Is(err, enums.ErrReqTerminalState) {
			r.state = to
			return customerrors.ReqFailedErr{
				Code:    http.StatusBadRequest,
				Message: fmt.Sprintf("terminal state reached: %s", to),
				Status:  to,
			}
		}
		return err
	}
	r.state = to
	return nil
}

func (r *request) hasReachedTerminalState() bool {
	return r.state.IsTerminalState(r.state)
}

type HandleRequestData struct {
	ConfigLoader            config.Loader
	Validator               validate.Validator
	Proxier                 proxy.Proxier
	RateLimiter             ratelimit.RateLimiter
	ServiceAccessController controller.ServiceAccessController
	GetService              services.GetServiceFunc
}

func GetHandlerFunc(hrd HandleRequestData) (http.HandlerFunc, error) {

	config, err := hrd.ConfigLoader.Load()
	if err != nil {
		return nil, errors.New("failed to load config")
	}

	handleTerminalReqFailure := func(w http.ResponseWriter, err error) bool {
		var reqFailedError customerrors.ReqFailedErr
		if errors.As(err, &reqFailedError) {
			w.WriteHeader(reqFailedError.Code)
			w.Write([]byte(reqFailedError.Message))
			return true
		}
		return false
	}

	var storer = hrd.GetService(config.Services)

	return func(w http.ResponseWriter, r *http.Request) {

		service, err := storer.Map(r.URL.Path)
		if handleTerminalReqFailure(w, err) {
			return
		}

		req := NewRequest()
		err = req.NextState(enums.ReqServiceMapSuccess)
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = hrd.ServiceAccessController.Control(controller.AccessControllerOpts{
			RestrictedServices: config.InternalOnlyServices,
			RestrictedPaths:    config.InternalOnlyServicesUrls,
			ServiceName:        service.ServiceName,
			ReqPath:            r.URL.Path,
		})
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = hrd.Validator.Validate(validate.ValidationOpts{
			GlobalBlockedIps:   config.BlockedIps,
			GlobalReqSizeLimit: config.ReqSizeLimit,
			ServiceOpts:        service,
		})
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = req.NextState(enums.ReqValidationSuccess)
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = hrd.RateLimiter.Limit(ratelimit.RateLimitOpts{
			GlobalRouteLimits:    config.RateLimit,
			ServiceRateLimitOpts: *service.RateLimitOpts,
		})
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = req.NextState(enums.ReqRateLimitSuccess)
		if handleTerminalReqFailure(w, err) {
			return
		}

		res, err := hrd.Proxier.Proxy(r, *service.RedirectOpts)
		if handleTerminalReqFailure(w, err) {
			return
		}

		err = req.NextState(enums.ReqProxySuccess)
		if handleTerminalReqFailure(w, err) {
			return
		}

		for k, v := range res.Header {
			w.Header().Set(k, strings.Join(v, ","))
		}
		body, err := io.ReadAll(res.Body)
		w.WriteHeader(res.StatusCode)
		w.Write(body)

		err = req.NextState(enums.ReqSuccess)
		if handleTerminalReqFailure(w, err) {
			return
		}
	}, nil

}
