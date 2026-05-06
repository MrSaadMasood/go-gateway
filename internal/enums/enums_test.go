package enums

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequest(t *testing.T) {

	tt := []struct {
		from    RequestStatus
		to      RequestStatus
		isError bool
		title   string
	}{
		{
			from:    ReqProxySuccess,
			to:      ReqInitialized,
			isError: true,
			title:   "should not go from intermediate state to initial state",
		},
		{
			from:    ReqInitialized,
			to:      ReqInitialized,
			isError: true,
			title:   "current and next state cannot be equal",
		},
		{
			from:    ReqFailed,
			to:      ReqProxySuccess,
			isError: true,
			title:   "should not go from terminal failed state to an intermediate state",
		},
		{
			from:    ReqSuccess,
			to:      ReqRateLimitFailed,
			isError: true,
			title:   "should not go from terminal success state to an intermediate state",
		},
		{
			from:    ReqInitialized,
			to:      ReqRateLimitFailed,
			isError: true,
			title:   "should skip any step and move to the next state from the initial state",
		},
		{
			from:    ReqProxyFailed,
			to:      ReqSuccess,
			isError: true,
			title:   "should not allow an invalid intermediate state",
		},
		{
			from:    ReqValidationSuccess,
			to:      ReqRateLimitSuccess,
			isError: false,
			title:   "should correctly move the state to the next in line one",
		},
		{
			from:    ReqRateLimitSuccess,
			to:      ReqValidationSuccess,
			isError: true,
			title:   "should not allow moving to the previous intermediate state",
		},
		{
			from:    ReqTimeout,
			to:      ReqSuccess,
			isError: true,
			title:   "timeout should not result in request success",
		},
		{
			from:    ReqTimeout,
			to:      ReqFailed,
			isError: true,
			title:   "timeout should not result in any other failure state",
		},
		{
			from:    ReqValidationSuccess,
			to:      ReqTimeout,
			isError: false,
			title:   "request can timeout at any step",
		},
		{
			from:    ReqValidationFailed,
			to:      ReqProxySuccess,
			isError: true,
			title:   "req validation failed should be a terminal state",
		},
		{
			from:    ReqProxyFailed,
			to:      ReqRateLimitSuccess,
			isError: true,
			title:   "req proxy failed should be a terminal state",
		},
	}

	var rs RequestStatus

	for _, data := range tt {
		t.Run(data.title, func(t *testing.T) {
			err := rs.Next(data.from, data.to)
			assert.Equal(t, data.isError, err != nil)
		})
	}

}
