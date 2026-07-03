package config

import (
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/stretchr/testify/assert"
)

func TestConfig(t *testing.T) {
	v := validator.New()

	testTables := []struct {
		name string
		t    func(t *testing.T)
	}{
		{
			name: "should validate the config",
			t: func(t *testing.T) {
				c := Config{
					Port:                         5000,
					GlobalTimeoutInSeconds:       10,
					RateLimitPerMinute:           10,
					ReqSizeLimitInBytes:          100,
					BlockedIps:                   []string{},
					AllowedOrigins:               []string{},
					HealthCheckIntervalInSeconds: 2,
					Services:                     []ServiceConfig{{ServiceName: "test-service", ServiceUrl: "https://test-service.com", AuthOpts: ServcieAuthOpts{}}},
				}

				err := v.Struct(c)
				assert.NoError(t, err)
			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}
}
