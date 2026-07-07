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
					Services:                     []ServiceConfig{{ServiceName: "test-service", ServiceUrl: "https://test-service.com", AuthOpts: ServcieAuthOpts{}, TimeoutInSeconds: nil, RateLimitOpts: nil}},
				}

				err := v.Struct(c)
				assert.NoError(t, err)

			},
		},
		{
			name: "should validate core service config",
			t: func(t *testing.T) {
				sc := ServiceConfig{
					ServiceName: "test-service", ServiceUrl: "https://test-service.com", AuthOpts: ServcieAuthOpts{}, TimeoutInSeconds: nil, RateLimitOpts: nil}

				err := v.Struct(sc)
				assert.NoError(t, err)

				timeout := 1.0
				sc.TimeoutInSeconds = &timeout

				err = v.Struct(sc)
				assert.NoError(t, err)
			},
		},
		{
			name: "should test core service rate limit options",
			t: func(t *testing.T) {
				rlo := ServiceRateLimitOpts{
					RateLimitPerMinute:            nil,
					RouteLevelRateLimitsPerMinute: nil,
				}

				err := v.Struct(rlo)
				assert.NoError(t, err)

				rateLimit := 1.0
				rlo.RateLimitPerMinute = &rateLimit
				rlo.RouteLevelRateLimitsPerMinute = make(map[string]float64)

				err = v.Struct(rlo)
				assert.NoError(t, err)

				rlo.RouteLevelRateLimitsPerMinute["endpoint"] = 12
				err = v.Struct(rlo)
				assert.NoError(t, err)

			},
		},
		{
			name: "should test core service redirect options",
			t: func(t *testing.T) {

				ro := ServiceRedirectOpts{
					RouteLevelRedirection:    nil,
					ProxyReqTimeoutInSeconds: nil,
				}

				err := v.Struct(ro)
				assert.NoError(t, err)

				timeout := 1.0
				ro.ProxyReqTimeoutInSeconds = &timeout
				ro.RouteLevelRedirection = make(map[ReqPath]redirectPath)

				err = v.Struct(ro)
				assert.NoError(t, err)

				ro.RouteLevelRedirection["endpoint"] = "redirected"

				err = v.Struct(ro)
				assert.NoError(t, err)
			},
		},
		{
			name: "should test core service validator options",
			t: func(t *testing.T) {
				vo := ServiceValidatorOpts{
					RequiredHeaders:   nil,
					RestrictedHeaders: nil,
					AllowedHeaders:    nil,
				}

				err := v.Struct(vo)
				assert.NoError(t, err)

				vo.RequiredHeaders = []string{""}
				err = v.Struct(vo)
				assert.Error(t, err)

				vo.RequiredHeaders = []string{"string"}

				err = v.Struct(vo)
				assert.NoError(t, err)

				vo.AllowedHeaders = []string{""}
				vo.RestrictedHeaders = []string{""}
				err = v.Struct(vo)
				assert.Error(t, err)

				vo.AllowedHeaders = []string{"header-2"}
				vo.RestrictedHeaders = []string{"header-3"}
				err = v.Struct(vo)
				assert.NoError(t, err)

			},
		},

		{
			name: "should test core service deprecation options",
			t: func(t *testing.T) {
				do := ServiceDepricationOpts{
					DeprecatedUrls:    nil,
					DeprecatedHeaders: nil,
					ObsoleteUrls:      nil,
				}

				err := v.Struct(do)
				assert.NoError(t, err)

				do.DeprecatedUrls = []ReqPath{""}
				err = v.Struct(do)
				assert.Error(t, err)

				do.DeprecatedUrls = []ReqPath{"string"}

				err = v.Struct(do)
				assert.NoError(t, err)

				do.DeprecatedHeaders = []string{""}
				do.ObsoleteUrls = []ReqPath{""}
				err = v.Struct(do)
				assert.Error(t, err)

				do.DeprecatedHeaders = []string{"header-2"}
				do.ObsoleteUrls = []ReqPath{"header-3"}
				err = v.Struct(do)
				assert.NoError(t, err)

			},
		},
		{
			name: "should test core service version options",
			t: func(t *testing.T) {
				vo := ServiceVersionOpts{
					AvialableVersions: nil,
					DefaultVersion:    "",
				}

				err := v.Struct(vo)
				assert.NoError(t, err)

				vo.AvialableVersions = []string{""}
				err = v.Struct(vo)
				assert.Error(t, err)

				vo.AvialableVersions = []string{"v1"}
				err = v.Struct(vo)
				assert.NoError(t, err)

			},
		},
		{
			name: "should validate core service policy options",
			t: func(t *testing.T) {
				po := ServicePolicyOpts{}
				err := v.Struct(po)
				assert.NoError(t, err)
			},
		},
		{
			name: "should validate core service blocked ips options",
			t: func(t *testing.T) {
				bio := ServiceBlockedIpsOpts{BlockedIps: nil, RouteLevelBlockedIps: nil}

				err := v.Struct(bio)
				assert.NoError(t, err)

				bio.BlockedIps = []string{""}
				err = v.Struct(bio)
				assert.Error(t, err)

				bio.BlockedIps = []string{"ip"}
				err = v.Struct(bio)
				assert.NoError(t, err)

				bio.RouteLevelBlockedIps = map[string][]string{}
				err = v.Struct(bio)
				assert.NoError(t, err)

				bio.RouteLevelBlockedIps["route"] = []string{""}
				err = v.Struct(bio)
				assert.Error(t, err)

				bio.RouteLevelBlockedIps["route"] = []string{"ip"}
				err = v.Struct(bio)
				assert.NoError(t, err)
			},
		},
		{
			name: "shoudl test core req origin options",
			t: func(t *testing.T) {
				roo := ServiceReqOriginOpts{
					AllowedOrigins: nil,
				}

				err := v.Struct(roo)
				assert.NoError(t, err)

				roo.AllowedOrigins = []string{""}
				err = v.Struct(roo)
				assert.Error(t, err)

				roo.AllowedOrigins = []string{"origin"}
				err = v.Struct(roo)
				assert.NoError(t, err)

			},
		},
		{
			name: "should test core bearer token policy options",
			t: func(t *testing.T) {
				btpo := ServiceBearerTokenPolicyOpts{
					ShouldVerifyBearerToken:   false,
					SkipBearerTokenCheckPaths: nil,
				}

				err := v.Struct(btpo)
				assert.NoError(t, err)

				btpo.SkipBearerTokenCheckPaths = []string{""}
				err = v.Struct(btpo)
				assert.Error(t, err)

				btpo.SkipBearerTokenCheckPaths = []string{"path-1"}
				err = v.Struct(btpo)
				assert.NoError(t, err)
			},
		},
		{
			name: "should panic if the config file is not present at the give address",
			t: func(t *testing.T) {
				c := NewConfigLoader("./test-configs/fake-config.json")
				assert.Panics(t, func() {
					c.Load()
				})
			},
		},
		{
			name: "should parse the file if the config file is present at the given address",
			t: func(t *testing.T) {

				c := NewConfigLoader("./test-configs/config.json")
				assert.NotPanics(t, func() {
					c.Load()
				})
			},
		},
	}

	for _, test := range testTables {
		t.Run(test.name, test.t)
	}
}
