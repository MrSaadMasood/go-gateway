package config

import (
	"strings"
	"time"
)

type ReqPath string
type redirectPath string

type ServiceConfigMap map[string]ServiceConfig

type ServiceRateLimitOpts struct {
	RateLimit            *float64
	RouteLevelRateLimits map[string]float64
}

type ServiceBlockedIpsOpts struct {
	BlockedIps           []string
	RouteLevelBlockedIps map[string][]string
}

type ServiceReqOriginOpts struct {
	AllowedOrigins []string
}

type ServiceBearerTokenPolicyOpts struct {
	ShouldVerifyBearerToken   bool
	SkipBearerTokenCheckPaths []string
}

type ServicePolicyOpts struct {
	*ServiceBlockedIpsOpts
	*ServiceReqOriginOpts
	*ServiceBearerTokenPolicyOpts
}

type ServiceValidatorOpts struct {
	RequiredHeaders   []string
	RestrictedHeaders []string
	AllowedHeaders    []string
}

type ServiceRedirectOpts struct {
	RouteLevelRedirection map[ReqPath]redirectPath
	ProxyReqTimeout       *time.Duration
}

type ServiceVersionOpts struct {
	AvialableVersions []string
	DefaultVersion    string
}

type ServiceDepricationOpts struct {
	DeprecatedUrls    []ReqPath
	DeprecatedHeaders []string
	ObsoleteUrls      []ReqPath
}

type ServcieAuthOpts struct {
	ValidatorOpts   *ServiceValidatorOpts
	DeprecationOpts *ServiceDepricationOpts
	PolicyOpts      *ServicePolicyOpts
	VersionOpts     *ServiceVersionOpts
}

type ServiceConfig struct {
	ServiceName   string
	ServiceUrl    string
	Timeout       *time.Duration
	RateLimitOpts *ServiceRateLimitOpts
	RedirectOpts  *ServiceRedirectOpts
	AuthOpts      ServcieAuthOpts
}

func NewServiceConfig(name, url string, timeout *time.Duration, rateLimitOpts *ServiceRateLimitOpts, redirectOpts *ServiceRedirectOpts, authOpts ServcieAuthOpts) (ServiceConfig, error) {
	if !strings.HasPrefix(url, "/") {
		return ServiceConfig{}, nil
	}

	return ServiceConfig{
		ServiceName:   name,
		ServiceUrl:    url,
		Timeout:       timeout,
		RateLimitOpts: rateLimitOpts,
		RedirectOpts:  redirectOpts,
		AuthOpts:      authOpts,
	}, nil
}

func (sc *ServiceConfig) GetProxyTimeout(globalTimeout time.Duration) time.Duration {

	var timeout time.Duration
	if sc.RedirectOpts != nil && sc.RedirectOpts.ProxyReqTimeout != nil {
		timeout = *sc.RedirectOpts.ProxyReqTimeout
	} else if sc.Timeout != nil {
		timeout = *sc.Timeout
	} else {
		timeout = globalTimeout
	}
	return timeout

}

type Config struct {
	Port                     int
	Timeout                  time.Duration
	RateLimit                float64
	Services                 []ServiceConfig
	InternalOnlyServices     []string
	InternalOnlyServicesUrls []string

	ReqSizeLimitInBytes int
	BlockedIps          []string
	AllowedOrigins      []string

	HealthCheckInterval time.Duration
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
	return Config{
		Port:                     5000,
		Timeout:                  10 * time.Second,
		RateLimit:                30,
		ReqSizeLimitInBytes:      5000,
		Services:                 nil,
		InternalOnlyServices:     nil,
		InternalOnlyServicesUrls: nil,
		BlockedIps:               nil,
		HealthCheckInterval:      10 * time.Second,
	}, nil
}

func NewMockConfigLoader() *ConfigLoader {
	return &ConfigLoader{}
}

type Loader interface {
	Load() (Config, error)
}
