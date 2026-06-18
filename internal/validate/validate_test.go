package validate

import (
	"gateway/internal/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

type mockConfig struct {
	mock.Mock
}

func (mc *mockConfig) Load() (config.Config, error) {
	args := mc.Called()
	return args.Get(0).(config.Config), nil
}

func TestValidate(t *testing.T) {

	tt := []struct {
		name string
		t    func(t *testing.T)
	}{
		{

			name: " should allow only valid ip adress format to be processed",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						DeprecationOpts: nil,
						PolicyOpts:      nil,
						VersionOpts:     nil,
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)

				r.RemoteAddr = "1.1"
				err := v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.RemoteAddr = "1.1.1.1:5000"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

			},
		},
		{

			name: " should only allow requests to known services ",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						DeprecationOpts: nil,
						PolicyOpts:      nil,
						VersionOpts:     nil,
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				unRegisteredServcie := "not-registered-service"
				inValidEndpoint := "/" + unRegisteredServcie + "/v1"
				validEndpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, inValidEndpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)

				err := v.Validate(w, r, unRegisteredServcie)
				assert.Error(t, err)

				vr := httptest.NewRequest(http.MethodGet, validEndpoint, http.NoBody)
				defer r.Body.Close()
				err = v.Validate(w, vr, testService1.ServiceName)
				assert.NoError(t, err)

			},
		},
		{

			name: " should block request ip addresses based on the configuration ",
			t: func(t *testing.T) {

				globalBlockedIp := "1.1.1.1"
				serviceBlockedIp := "2.2.2.2"
				routeBlockedIp := "3.3.3.3"

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						DeprecationOpts: nil,
						PolicyOpts: &config.ServicePolicyOpts{
							ServiceBlockedIpsOpts: &config.ServiceBlockedIpsOpts{
								BlockedIps: []string{serviceBlockedIp},
								RouteLevelBlockedIps: map[string][]string{
									"/": {routeBlockedIp},
								},
							},
						},
						VersionOpts: nil,
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{globalBlockedIp}, []string{}, 12, scm)

				r.RemoteAddr = globalBlockedIp + ":3000"
				err := v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.RemoteAddr = serviceBlockedIp + ":3000"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.RemoteAddr = routeBlockedIp + ":3000"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.RemoteAddr = "0.0.0.0:7000"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

			},
		},
		{

			name: "should validate the req headers",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts: &config.ServiceValidatorOpts{
							RequiredHeaders:   []string{"Required-1"},
							RestrictedHeaders: []string{"Restricted-1"},
							AllowedHeaders:    []string{"Allowed-1"},
						},
						DeprecationOpts: &config.ServiceDepricationOpts{
							DeprecatedHeaders: []string{"Deprecated-1"},
						},
						VersionOpts: nil,
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)

				r.Header.Add("Required-1", "value")
				err := v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				r.Header.Add("Restricted-1", "value")
				err = v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.Header.Del("Restricted-1")
				r.Header.Add("Deprecated-1", "value")
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				deprecatedResHeader := w.Header().Get("Deprecation")
				warningResHeader := w.Header().Get("Warning")
				assert.Greater(t, len(deprecatedResHeader), 0, "the deprecation header should not be nil")
				assert.Greater(t, len(warningResHeader), 0, "the deprecation header should not be nil")

				r.Header.Add("Allowed-1", "value")
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

			},
		},
		{

			name: "should validate the bearer token if enabled",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						PolicyOpts: &config.ServicePolicyOpts{
							ServiceBearerTokenPolicyOpts: &config.ServiceBearerTokenPolicyOpts{
								ShouldVerifyBearerToken:   true,
								SkipBearerTokenCheckPaths: []string{"/default"},
							},
						},
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)
				err := v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

				r.Header.Add("authorization", "Bearer token")
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				r.URL.Path = testService1.ServiceUrl + "/v1/default"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				testService1.AuthOpts.PolicyOpts.ServiceBearerTokenPolicyOpts.ShouldVerifyBearerToken = false
				r.URL.Path = testService1.ServiceUrl + "/v1/new"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

			},
		},
		{

			name: "should validate the deprecated paths",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						DeprecationOpts: &config.ServiceDepricationOpts{
							DeprecatedUrls: []config.ReqPath{"/deprecated-1"},
							ObsoleteUrls:   []config.ReqPath{"/obsolete-1"},
						},
						VersionOpts: nil,
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1/deprecated-1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)

				err := v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				deprecatedResHeader := w.Header().Get("Deprecation")
				warningResHeader := w.Header().Get("Warning")

				assert.Greater(t, len(deprecatedResHeader), 0, "the deprecation header should not be nil")
				assert.Greater(t, len(warningResHeader), 0, "the deprecation header should not be nil")

				r.URL.Path = testService1.ServiceUrl + "/v1/obsolete-1"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

			},
		},
		{

			name: "should validate the version of the service request",
			t: func(t *testing.T) {

				testService1 := config.ServiceConfig{
					ServiceName:   "test-service",
					ServiceUrl:    "/test-service",
					Timeout:       nil,
					RateLimitOpts: nil,
					RedirectOpts:  nil,
					AuthOpts: config.ServcieAuthOpts{
						ValidatorOpts:   nil,
						DeprecationOpts: nil,
						PolicyOpts:      nil,
						VersionOpts: &config.ServiceVersionOpts{
							AvialableVersions: []string{"v1", "v2", "v4"},
						},
					},
				}

				scm := map[string]config.ServiceConfig{
					testService1.ServiceName: testService1,
				}

				endpoint := testService1.ServiceUrl + "/v1"

				r := httptest.NewRequest(http.MethodGet, endpoint, http.NoBody)
				defer r.Body.Close()
				w := httptest.NewRecorder()

				v := NewReqValidator([]string{}, []string{}, 12, scm)
				err := v.Validate(w, r, testService1.ServiceName)
				assert.NoError(t, err)

				r.URL.Path = testService1.ServiceUrl + "/v3"
				err = v.Validate(w, r, testService1.ServiceName)
				assert.Error(t, err)

			},
		},
	}

	for _, test := range tt {
		t.Run(test.name, test.t)
	}
}
