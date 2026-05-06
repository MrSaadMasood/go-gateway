package customerrors

import (
	"fmt"
	"gateway/internal/enums"
)

type ReqFailedErr struct {
	Code    int
	Message string
	Status  enums.RequestStatus
}

func (rfe ReqFailedErr) Error() string {
	return fmt.Sprintf("message: %s", rfe.Message)
}
