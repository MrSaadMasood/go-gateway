package request

import (
	"context"
	"errors"
	"fmt"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net/http"
	"slices"
)

type reqEventStatus struct {
	event enums.ReqEvent
	from  enums.RequestStatus
}

type reqTransitionKeyActionMap map[reqEventStatus]func(r *http.Request) error

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

func (m *reqStateMachine) initializeRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		newReq := r.WithContext(WithReqStatus(ctx, enums.ReqInitialized))
		h.ServeHTTP(w, newReq)
	})
}

func (m *reqStateMachine) moveToMapSuccess(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req, err := m.fire(r, enums.MapSuccessReqEvent, enums.MapFailedReqEvent)
		if err != nil {

		}
		h.ServeHTTP(w, req)
	})
}

func (m *reqStateMachine) fire(r *http.Request, successEvent enums.ReqEvent, failed enums.ReqEvent) (*http.Request, error) {
	ctx := r.Context()

	from, err := StatusFrom(ctx)
	if err != nil {
		return nil, fmt.Errorf("status from context not found")
	}

	sf, ok := m.eventStateMap[successEvent]
	if !ok {
		return nil, fmt.Errorf("invalid event provided: %s", successEvent)
	}

	if !slices.Contains(sf.fromStatus, from) {
		return nil, fmt.Errorf("invalid request flow: %s", from)
	}

	key := GenerateActionKey(successEvent, from)
	action, ok := m.keyActionMap[key]
	if !ok {
		return nil, fmt.Errorf("action not found for key: %+v", key)
	}

	err = action(r)
	if err != nil {
		return nil, fmt.Errorf("error occured while performing action: %w", err)
	}

	return m.updateRequestStatus(r, sf.to), nil
}

func (m *reqStateMachine) requestFailedMiddleware(e customerrors.ReqFailedErr) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(e.Code)
		w.Write([]byte(e.Message))
	})
}

func (m *reqStateMachine) updateRequestStatus(r *http.Request, to enums.RequestStatus) *http.Request {
	ctx := r.Context()
	newReq := r.WithContext(WithReqStatus(ctx, to))
	return newReq
}

func GenerateActionKey(event enums.ReqEvent, from enums.RequestStatus) reqEventStatus {
	return reqEventStatus{
		event: event,
		from:  from,
	}
}

func WithReqStatus(ctx context.Context, status enums.RequestStatus) context.Context {
	return context.WithValue(ctx, "status", status)
}

func StatusFrom(ctx context.Context) (enums.RequestStatus, error) {
	s, ok := ctx.Value("status").(enums.RequestStatus)
	if !ok {
		return "", errors.New("status not found from context")
	}
	return s, nil
}
