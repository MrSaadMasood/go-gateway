package ratelimit

import (
	"context"
	"gateway/internal/config"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTockenBucket(t *testing.T) {
	tt := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: " should test the token bucket refill, refill cancellation and token consumption ",
			t: func(t *testing.T) {
				ctx, cancel := context.WithCancel(context.Background())
				tb := newTokenbucket(ctx, 2, 100*time.Millisecond)
				time := time.After(1 * time.Second)
				<-time
				cancel()
				count := 0
				for tb.getToken() != nil {
					count++
				}

				assert.Equal(t, tb.capacity, count)
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, test.t)
	}
}

func TestReqRateLimiter(t *testing.T) {

	ip := "0.0.0.0:3000"

	tt := []struct {
		name string
		t    func(t *testing.T)
	}{

		{
			name: " should rate limit properly at global level ",
			t: func(t *testing.T) {

				rateLimitOpts := config.ServiceRateLimitOpts{
					RateLimit:            nil,
					RouteLevelRateLimits: nil,
				}

				testService1 := config.ServiceConfig{
					ServiceName:        "test-service",
					ServiceUrl:         "/test-service",
					Timeout:            nil,
					RateLimitOpts:      &rateLimitOpts,
					ValidatorOpts:      nil,
					RedirectOpts:       nil,
					UrlDepricationOpts: nil,
					PolicyOpts:         nil,
					VersionOpts:        nil,
				}

				serviceConfigs := []config.ServiceConfig{
					testService1,
				}
				globalRl := 120
				cap, rate := calculateBucketData(globalRl)
				bufferedTime := time.After(rate + (5 * time.Second))
				endpoint := "/test-service/v1"

				rrl := NewReqRateLimiter(context.Background(), globalRl, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, endpoint, ip)
				}

				results := make([]error, 0)
				done := make(chan bool)

				go func() {
					ticker := time.NewTicker(rate / 2)
					for {
						select {
						case <-done:
							return
						case <-ticker.C:
							err := rrl.Limit(testService1.ServiceName, endpoint, ip)
							results = append(results, err)
						}
					}

				}()

				<-bufferedTime
				done <- true

				var successfullReq int
				var failedReq int

				for _, v := range results {
					if v == nil {
						successfullReq++
					} else {
						require.ErrorIs(t, v, globalRateLimitErr)
						failedReq++
					}
				}

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should rate limit at service level",
			t: func(t *testing.T) {

				servcieRl := 120
				rateLimitOpts := config.ServiceRateLimitOpts{
					RateLimit:            &servcieRl,
					RouteLevelRateLimits: nil,
				}

				testService1 := config.ServiceConfig{
					ServiceName:        "test-service",
					ServiceUrl:         "/test-service",
					Timeout:            nil,
					RateLimitOpts:      &rateLimitOpts,
					ValidatorOpts:      nil,
					RedirectOpts:       nil,
					UrlDepricationOpts: nil,
					PolicyOpts:         nil,
					VersionOpts:        nil,
				}

				serviceConfigs := []config.ServiceConfig{
					testService1,
				}

				cap, rate := calculateBucketData(servcieRl)
				bufferedTime := time.After(rate + (5 * time.Second))
				endpoint := "/test-service/v1"

				rrl := NewReqRateLimiter(context.Background(), 1000, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, endpoint, ip)
				}

				results := make([]error, 0)
				done := make(chan bool)

				go func() {
					ticker := time.NewTicker(rate / 2)
					for {
						select {
						case <-done:
							return
						case <-ticker.C:
							err := rrl.Limit(testService1.ServiceName, endpoint, ip)
							results = append(results, err)
						}
					}
				}()

				<-bufferedTime
				done <- true

				var successfullReq int
				var failedReq int

				for _, v := range results {
					if v == nil {
						successfullReq++
					} else {
						require.ErrorIs(t, v, serviceLevelRateLimitErr)
						failedReq++
					}
				}

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should rate limit at the route level",
			t: func(t *testing.T) {

				servcieRl := 500
				routeRl := 120
				rateLimitOpts := config.ServiceRateLimitOpts{
					RateLimit: &servcieRl,
					RouteLevelRateLimits: map[string]int{
						"/v1": routeRl,
					},
				}

				serviceUrl := "/test-service"
				testService1 := config.ServiceConfig{
					ServiceName:        "test-service",
					ServiceUrl:         serviceUrl,
					Timeout:            nil,
					RateLimitOpts:      &rateLimitOpts,
					ValidatorOpts:      nil,
					RedirectOpts:       nil,
					UrlDepricationOpts: nil,
					PolicyOpts:         nil,
					VersionOpts:        nil,
				}

				serviceConfigs := []config.ServiceConfig{
					testService1,
				}

				globalRl := 1000
				cap, rate := calculateBucketData(routeRl)
				bufferedTime := time.After(rate + (5 * time.Second))

				rrl := NewReqRateLimiter(context.Background(), globalRl, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, "/v1", ip)
				}

				results := make([]error, 0)
				done := make(chan bool)

				go func() {
					ticker := time.NewTicker(rate / 2)
					for {
						select {
						case <-done:
							return
						case <-ticker.C:
							err := rrl.Limit(testService1.ServiceName, "/v1", ip)
							results = append(results, err)
						}
					}
				}()

				<-bufferedTime
				done <- true

				var successfullReq int
				var failedReq int

				for _, v := range results {
					if v == nil {
						successfullReq++
					} else {
						require.ErrorIs(t, v, routeLevelRateLimitErr)
						failedReq++
					}
				}

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should ensure that req limiter treats different tips as separate entities / users",
			t: func(t *testing.T) {

			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, test.t)
	}

}
