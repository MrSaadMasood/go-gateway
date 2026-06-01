package customerrors

import (
	"context"
	"errors"
	"gateway/internal/enums"
	"net/http"
)

type ReqFailer interface {
	Error() string
	Code() int
	Status() enums.RequestStatus
}
type reqFailedErr struct {
	code   int
	status enums.RequestStatus
	error
}

func NewReqFailedErr(code int, status enums.RequestStatus, err error) error {
	if errors.Is(err, context.DeadlineExceeded) {
		code = http.StatusRequestTimeout
		status = enums.ReqTimeout
	}

	return reqFailedErr{
		code:   code,
		status: status,
		error:  err,
	}
}

func (rfe reqFailedErr) Error() string {
	return rfe.error.Error()
}

func (rfe reqFailedErr) Code() int {
	return rfe.code
}

func (rfe reqFailedErr) Status() enums.RequestStatus {
	return rfe.status
}
