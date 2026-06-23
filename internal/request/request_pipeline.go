package request

import (
	"context"
	"errors"
	"fmt"
	"gateway/internal/common"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"gateway/internal/log"
	"gateway/internal/types"
	"net/http"
	"slices"
)

type reqEventStatus struct {
	event enums.ReqEvent
	from  enums.RequestStatus
}

type ActionFunc func(w http.ResponseWriter, r *http.Request) (*http.Request, error)

type reqTransitionKeyActionMap map[reqEventStatus]ActionFunc

type reqStatusFlow struct {
	fromStatus []enums.RequestStatus
	to         enums.RequestStatus
}

type reqStateMachine struct {
	eventStateMap map[enums.ReqEvent]reqStatusFlow
	keyActionMap  reqTransitionKeyActionMap
}

func NewReqStateMachine(kam reqTransitionKeyActionMap) reqStateMachine {
	return reqStateMachine{
		keyActionMap: kam,
		eventStateMap: map[enums.ReqEvent]reqStatusFlow{
			enums.MapSuccessReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqInitialized},
				to:         enums.ReqServiceMapSuccess,
			},
			enums.MapFailedReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqInitialized},
				to:         enums.ReqServiceMapFailed,
			},
			enums.ValidationSuccessReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqServiceMapSuccess},
				to:         enums.ReqValidationSuccess,
			},
			enums.ValidationFailedReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqServiceMapSuccess},
				to:         enums.ReqValidationFailed,
			},
			enums.RateLimitSuccessReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqValidationSuccess},
				to:         enums.ReqRateLimitSuccess,
			},
			enums.RateLimitFailedReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqValidationSuccess},
				to:         enums.ReqRateLimitFailed,
			},
			enums.ProxySuccessReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqRateLimitSuccess},
				to:         enums.ReqProxySuccess,
			},
			enums.ProxyFailedReqEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqRateLimitSuccess},
				to:         enums.ReqProxyFailed,
			},
			enums.ReqSuccessEvent: {
				fromStatus: []enums.RequestStatus{enums.ReqProxySuccess},
				to:         enums.ReqSuccess,
			},
			enums.ReqFailedEvent: {
				fromStatus: []enums.RequestStatus{
					enums.ReqServiceMapFailed,
					enums.ReqValidationFailed,
					enums.ReqRateLimitFailed,
					enums.ReqProxyFailed,
					enums.ReqTimeout,
				},
				to: enums.ReqSuccess,
			},
			enums.ReqTimeoutEvent: {
				fromStatus: []enums.RequestStatus{
					enums.ReqInitialized,
					enums.ReqServiceMapSuccess,
					enums.ReqServiceMapFailed,
					enums.ReqValidationSuccess,
					enums.ReqValidationFailed,
					enums.ReqRateLimitSuccess,
					enums.ReqRateLimitFailed,
					enums.ReqProxySuccess,
					enums.ReqProxyFailed,
				},
				to: enums.ReqFailed,
			},
		},
	}
}

func (m *reqStateMachine) initializeRequest(f types.HandlerFuncWithError) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		newReq := r.WithContext(common.WithReqStatus(ctx, enums.ReqInitialized))
		err := f(w, newReq)
		sendError(err, w, r)
	})
}

func (m *reqStateMachine) serviceMapM(f types.HandlerFuncWithError) types.HandlerFuncWithError {
	return m.withWrapper(f, enums.MapSuccessReqEvent, enums.ReqServiceMapFailed)

}

func (m *reqStateMachine) validationM(f types.HandlerFuncWithError) types.HandlerFuncWithError {
	return m.withWrapper(f, enums.ValidationSuccessReqEvent, enums.ReqValidationFailed)
}

func (m *reqStateMachine) rateLimiterM(f types.HandlerFuncWithError) types.HandlerFuncWithError {
	return m.withWrapper(f, enums.RateLimitSuccessReqEvent, enums.ReqRateLimitFailed)
}

func (m *reqStateMachine) proxyM(f types.HandlerFuncWithError) types.HandlerFuncWithError {
	return m.withWrapper(f, enums.ProxySuccessReqEvent, enums.ReqProxyFailed)
}

func (m *reqStateMachine) sendSuccessM() types.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		_, err := m.fire(w, r, enums.ReqSuccessEvent, enums.ReqFailed)
		if err != nil {
			return err
		}
		return nil
	}
}

func (m *reqStateMachine) withWrapper(f types.HandlerFuncWithError, event enums.ReqEvent, fs enums.RequestStatus) types.HandlerFuncWithError {
	return m.customErrorHandlerM(func(w http.ResponseWriter, r *http.Request) error {

		req, err := m.fire(w, r, event, fs)
		if err != nil {
			return err
		}

		err = f(w, req)
		if err != nil {
			return err
		}

		return nil

	})
}

func (m *reqStateMachine) customErrorHandlerM(f types.HandlerFuncWithError) types.HandlerFuncWithError {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := f(w, r)
		sendError(err, w, r)
		return nil
	}
}

func (m *reqStateMachine) fire(w http.ResponseWriter, r *http.Request, successEvent enums.ReqEvent, failureState enums.RequestStatus) (*http.Request, error) {
	ctx := r.Context()

	from, err := common.StatusFrom(ctx)
	if err != nil {
		return nil, customerrors.NewReqFailedErr(http.StatusBadRequest, failureState, err)
	}

	sf, ok := m.eventStateMap[successEvent]
	if !ok {
		return nil, customerrors.NewReqFailedErr(http.StatusBadRequest, failureState, err)
	}

	if !slices.Contains(sf.fromStatus, from) {
		return nil, customerrors.NewReqFailedErr(http.StatusBadRequest, failureState, err)
	}

	key := GenerateActionKey(successEvent, from)
	action, ok := m.keyActionMap[key]
	if !ok {
		return nil, customerrors.NewReqFailedErr(http.StatusBadRequest, failureState, err)
	}

	req, err := action(w, r)
	if err != nil {
		return nil, customerrors.NewReqFailedErr(http.StatusBadRequest, failureState, err)
	}

	return m.updateRequestStatus(req, sf.to), nil
}

func (m *reqStateMachine) updateRequestStatus(r *http.Request, to enums.RequestStatus) *http.Request {
	ctx := r.Context()
	newReq := r.WithContext(common.WithReqStatus(ctx, to))
	return newReq
}

func GenerateActionKey(event enums.ReqEvent, from enums.RequestStatus) reqEventStatus {
	return reqEventStatus{
		event: event,
		from:  from,
	}
}

func WithProxyRes(ctx context.Context, res *http.Response) context.Context {
	return context.WithValue(ctx, "proxy-response", res)
}

func ProxyResFrom(ctx context.Context) (*http.Response, error) {
	res, ok := ctx.Value("proxy-response").(*http.Response)
	if !ok {
		return nil, errors.New("no response found from context")
	}
	return res, nil
}

func sendError(err error, w http.ResponseWriter, r *http.Request) {
	if err != nil {

		ld, logDataErr := common.LogDataFrom(r.Context())
		fd := log.FailureData{
			Reason: err.Error(),
		}
		var reqStatus enums.RequestStatus
		var reqFailer customerrors.ReqFailer

		defer func() {
			if logDataErr != nil {
				return
			}
			ld.SetFailure(fd)
			ld.SetFinalReqStatus(reqStatus)
		}()

		ok := errors.As(err, &reqFailer)
		if ok {

			fd.Reason = reqFailer.Error()
			reqStatus = reqFailer.Status()

			http.Error(w, fmt.Sprintf("req failed with status: %s and error: %s", reqFailer.Status(), reqFailer.Error()), reqFailer.Code())
			return
		}

		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
}
