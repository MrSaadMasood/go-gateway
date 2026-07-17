package gateway

import (
	"context"
	"errors"
	"gateway/internal/config"
	"gateway/internal/mocks"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestGateway(t *testing.T) {

	testTable := []struct {
		name string
		t    func(*testing.T)
	}{
		{
			name: "should start gateway successfully",
			t: func(t *testing.T) {

				configService := config.ServiceConfig{
					ServiceName:      "test-service",
					TimeoutInSeconds: nil,
					RateLimitOpts:    nil,
					RoutingOpts:      nil,
					ReqProxyOpts:     nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						PolicyOpts:      nil,
						VersionOpts:     nil,
						DeprecationOpts: nil,
					},
				}

				globalTimeout := 10.0
				c := config.Config{
					Port:                   5000,
					GlobalTimeoutInSeconds: globalTimeout,
					RateLimitPerMinute:     10,
					ReqSizeLimitInBytes:    3000,
					Services: []config.ServiceConfig{
						configService,
					},
					BlockedIps:                   []string{},
					HealthCheckIntervalInSeconds: int64(globalTimeout),
				}

				ms := new(mocks.MockStore)
				mcl := new(mocks.MockConfigLoader)

				callOrder := make([]int, 0)
				addCall := func(i int) func(args mock.Arguments) {
					callOrder = append(callOrder, i)
					return func(args mock.Arguments) {
					}
				}

				ms.On("Initialize").Return(nil).Run(addCall(1))
				mcl.On("Load").Return(c, nil).Run(addCall(2))

				ctx := context.Background()
				g := NewGateway(ctx, mcl, ms)
				assert.NotPanics(t, func() {
					g.Start()
				})
				assert.Equal(t, []int{1, 2}, callOrder)
			},
		},
		{
			name: "should panic if any dependency returns error",
			t: func(t *testing.T) {

				ms := new(mocks.MockStore)
				mcl := new(mocks.MockConfigLoader)

				ms.On("Initialize").Return(errors.New("initialization failed error"))
				mcl.On("Load").Return(config.Config{}, errors.New("failed to load config"))

				ctx := context.Background()
				g := NewGateway(ctx, mcl, ms)
				assert.Panics(t, func() {
					g.Start()
				})
			},
		},
	}

	for _, test := range testTable {
		t.Run(test.name, test.t)
	}
}
