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
				tb := newTokenbucket(ctx, 2, 1, 100*time.Millisecond, cancel)
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

	ip1 := "0.0.0.0:3000"
	ip2 := "1.1.1.1:3000"

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
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: &rateLimitOpts,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						DeprecationOpts: nil,
						PolicyOpts:      nil,
						VersionOpts:     nil,
					},
				}

				serviceConfigs := []config.ServiceConfig{
					testService1,
				}
				globalRl := 100.0
				cap, _, _, _ := calculateBucketData(globalRl)
				endpoint := "/test-service/v1"

				ctx, cancel := context.WithCancel(context.Background())
				// immediately cancel the context to stop token refilling
				cancel()

				rrl := NewReqRateLimiter(ctx, globalRl, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, endpoint, ip1)
				}

				err := rrl.Limit(testService1.ServiceName, endpoint, ip1)

				var successfullReq int
				var failedReq int

				require.ErrorIs(t, err, globalRateLimitErr)

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should rate limit at service level",
			t: func(t *testing.T) {

				servcieRl := 120.0
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

				cap, _, _, _ := calculateBucketData(servcieRl)
				endpoint := "/test-service/v1"

				ctx, cancel := context.WithCancel(context.Background())
				// immediately cancel the context to stop token refilling
				cancel()

				rrl := NewReqRateLimiter(ctx, 1000, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, endpoint, ip1)
				}

				err := rrl.Limit(testService1.ServiceName, endpoint, ip1)
				require.ErrorIs(t, err, serviceLevelRateLimitErr)

				var successfullReq int
				var failedReq int

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should rate limit at the route level",
			t: func(t *testing.T) {

				servcieRl := 500.0
				routeRl := 120.0
				rateLimitOpts := config.ServiceRateLimitOpts{
					RateLimit: &servcieRl,
					RouteLevelRateLimits: map[string]float64{
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

				globalRl := 1000.0
				cap, _, _, _ := calculateBucketData(routeRl)

				ctx, cancel := context.WithCancel(context.Background())
				// immediately cancel the context to stop token refilling
				cancel()

				rrl := NewReqRateLimiter(ctx, globalRl, serviceConfigs)

				for range cap {
					rrl.Limit(testService1.ServiceName, "/v1", ip1)
				}

				err := rrl.Limit(testService1.ServiceName, "/v1", ip1)
				require.ErrorIs(t, err, routeLevelRateLimitErr)

				var successfullReq int
				var failedReq int

				assert.GreaterOrEqual(t, failedReq, successfullReq)

			},
		},
		{
			name: "should ensure that req limiter treats different ips as separate entities / users",
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
				globalRl := 100.0
				endpoint := "/test-service/v1"

				ctx, cancel := context.WithCancel(context.Background())
				// immediately cancel the context to stop token refilling
				cancel()

				rrl := NewReqRateLimiter(ctx, globalRl, serviceConfigs)

				rrl.Limit(testService1.ServiceName, endpoint, ip1)
				rrl.Limit(testService1.ServiceName, endpoint, ip2)

				tBucket1 := rrl.getIpTBucket(ip1)
				tBucket2 := rrl.getIpTBucket(ip2)

				assert.NotEqual(t, tBucket1, tBucket2, "the buckets should be different for each ip")
			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, test.t)
	}

}
