package customerrors

import (
	"gateway/internal/enums"
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

func NewReqFailedErr(code int, status enums.RequestStatus, err error) reqFailedErr {
	return reqFailedErr{
		code:   code,
		status: status,
		error:  err,
	}
}

func (rfe reqFailedErr) Error() string {
	return rfe.Error()
}

func (rfe reqFailedErr) Code() int {
	return rfe.code
}

func (rfe reqFailedErr) Status() enums.RequestStatus {
	return rfe.status
}
