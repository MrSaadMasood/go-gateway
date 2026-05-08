package request

import "gateway/internal/enums"

type ReqTransitionData struct {
	event      enums.ReqEvent
	fromStatus []enums.RequestStatus
}

type ReqTransitionKeyActionMap map[string]func()

type ReqStatusFlow struct {
	fromStatus []enums.RequestStatus
	to         enums.RequestStatus
}

type reqStateMachine struct {
	eventStateMap map[enums.ReqEvent]ReqStatusFlow
	keyActionMap  *ReqTransitionKeyActionMap
}

func NewReqStateMachine(kam *ReqTransitionKeyActionMap) reqStateMachine {
	return reqStateMachine{
		keyActionMap: kam,
		eventStateMap: map[enums.ReqEvent]ReqStatusFlow{
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
				fromStatus: []enums.RequestStatus{enums.ReqServiceMapFailed, enums.ReqValidationFailed, enums.ReqRateLimitFailed, enums.ReqProxyFailed, enums.ReqTimeout},
				to:         enums.ReqSuccess,
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
					enums.ReqProxyFailed},
				to: enums.ReqFailed,
			},
		},
	}
}
