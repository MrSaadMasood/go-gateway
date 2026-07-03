package config

import (
	"encoding/json"
	"os"
	"time"

	"github.com/go-playground/validator/v10"
)

type Loader interface {
	Load() (Config, error)
}

type ReqPath string
type redirectPath string
type ServiceConfigMap map[string]ServiceConfig

type ServiceRateLimitOpts struct {
	// service level rate limit
	RateLimitPerMinute *float64 `json:"rate_limit_per_min"`

	// service route level rate limits
	RouteLevelRateLimitsPerMinute map[string]float64 `json:"route_level_limit_per_min" validate:"omitempty,dive,required"`
}

type ServiceBlockedIpsOpts struct {
	// service level blocked ips
	BlockedIps []string `json:"blocked_ips" validate:"omitempty,dive,required"`

	// service route level blocked ips
	RouteLevelBlockedIps map[string][]string `json:"route_level_blocked_ips" validate:"omitempty,dive,required"`
}

type ServiceReqOriginOpts struct {
	// service level allowed origins
	AllowedOrigins []string `json:"allowed_origins" validate:"omitempty,dive,required"`
}

type ServiceBearerTokenPolicyOpts struct {
	// if tru would check if the bearer token is present in the request headers
	ShouldVerifyBearerToken bool `json:"should_verify_bearer_token"`

	// skips bearer token existance check for the provided paths of the service
	SkipBearerTokenCheckPaths []string `json:"skip_bearer_token_check_path" validate:"omitempty,dive,required"`
}

type ServicePolicyOpts struct {
	*ServiceBlockedIpsOpts        `json:"blocked_ip_opts" validate:"omitempty,dive"`
	*ServiceReqOriginOpts         `json:"req_origin_opts" validate:"omitempty,dive"`
	*ServiceBearerTokenPolicyOpts `json:"bearer_token_policy_opts" validate:"omitempty,dive"`
}

type ServiceValidatorOpts struct {
	// headers that should be present in a request
	RequiredHeaders []string `json:"required_headers" validate:"omitempty,dive,required"`

	// headers that are restricted in a request
	RestrictedHeaders []string `json:"restricted_headers" validate:"omitempty,dive,required"`

	// only headers that are allowed in a request.
	AllowedHeaders []string `json:"allowed_headers" validate:"omitempty,dive,required"`
}

type ServiceRedirectOpts struct {
	// redirect the request from a particular paths to other paths
	RouteLevelRedirection map[ReqPath]redirectPath `json:"route_level_redirection" validate:"omitempty,dive,required"`

	// timeout for the request is being proxied
	ProxyReqTimeoutInSeconds *float64 `json:"proxy_req_timeout_sec"`
}

type ServiceVersionOpts struct {

	// available versions of the service; must be in format like v<int>
	AvialableVersions []string `json:"available_versions" validate:"omitempty,dive,required"`

	// default version of the service
	DefaultVersion string `json:"default_version"`
}

type ServiceDepricationOpts struct {

	// service request paths that are legacy
	DeprecatedUrls []ReqPath `json:"deprecated_urls" validate:"omitempty,dive,required"`

	// service request headers that are legacy
	DeprecatedHeaders []string `json:"deprecated_headers" validate:"omitempty,dive,required"`

	// service request paths that no longer available and would result in request failure
	ObsoleteUrls []ReqPath `json:"obsolete_urls" validate:"omitempty,dive,required"`
}

type ServcieAuthOpts struct {
	ValidatorOpts   *ServiceValidatorOpts   `json:"validator_opts" validate:"omitempty,dive"`
	DeprecationOpts *ServiceDepricationOpts `json:"deprecation_opts" validate:"omitempty,dive"`
	PolicyOpts      *ServicePolicyOpts      `json:"policy_opts" validate:"omitempty,dive"`
	VersionOpts     *ServiceVersionOpts     `json:"version_opts" validate:"omitempty,dive"`
}

type ServiceConfig struct {
	// the service name used by gateway to identify the service from the request path
	ServiceName string `json:"service_name" validate:"required"`

	// complete service url where the request would be proxied
	ServiceUrl string `json:"service_url" validate:"required"`

	// service level request timeout
	TimeoutInSeconds *float64 `json:"timeout_sec"`

	RateLimitOpts *ServiceRateLimitOpts `json:"rate_limit_opts" validate:"omitempty,dive"`
	RedirectOpts  *ServiceRedirectOpts  `json:"redirect_opts" validate:"omitempty,dive"`
	AuthOpts      ServcieAuthOpts       `json:"auth_opts" vadlidate:"dive,required"`
}

func (sc *ServiceConfig) GetProxyTimeout(globalTimeout time.Duration) time.Duration {

	var timeout time.Duration
	if sc.RedirectOpts != nil && sc.RedirectOpts.ProxyReqTimeoutInSeconds != nil {
		timeout = time.Duration(*sc.RedirectOpts.ProxyReqTimeoutInSeconds)
	} else if sc.TimeoutInSeconds != nil {
		timeout = time.Duration(*sc.TimeoutInSeconds)
	} else {
		timeout = globalTimeout
	}
	return timeout

}

type Config struct {
	// the port at which gateway should listen to the requests
	Port int `json:"port" validate:"required"`

	// max time for reading the headers, body and writing the response
	// timeout is also used for the global request context that would be propagated to all the requests.
	// the global request timeout would be 2x the provided value
	// also would be used as a default value if no value for proxying request is provided
	GlobalTimeoutInSeconds float64 `json:"global_timeout_sec" validate:"required"`

	// used for global rate limiting value
	RateLimitPerMinute float64 `json:"rate_limit_per_min" validate:"required"`

	// service specific configs
	Services []ServiceConfig `json:"services" validate:"dive,required,min=1"`

	// max req size allowed
	ReqSizeLimitInBytes int `json:"req_size_in_bytes" validate:"required"`

	// global ips that should be blocked
	BlockedIps []string `json:"blocked_ips" validate:"omitempty,dive,required"`

	// globally permitted origins
	AllowedOrigins []string `json:"allowed_origins" validate:"omitempty,dive,required"`

	// interval after which gateway would check the health of the services
	HealthCheckIntervalInSeconds int64 `json:"health_interval_sec" validate:"required"`
}

func (c *Config) GetServiceConfigMap() ServiceConfigMap {

	scm := make(map[string]ServiceConfig)

	for _, s := range c.Services {
		scm[s.ServiceName] = s
	}
	return scm
}

type ConfigLoader struct{}

func (ml *ConfigLoader) Load() (Config, error) {

	f, err := os.ReadFile("../../config.json")
	if err != nil {
		return Config{}, nil
	}

	var config Config
	err = json.Unmarshal(f, &config)
	if err != nil {
		panic(err)
	}

	v := validator.New()

	err = v.Struct(config)
	if err != nil {
		panic(err)
	}

	return config, nil

}

func NewConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}
