package config

import (
	"time"
)

type reqPath string
type redirectPath string

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

type ServiceReqAuthPolicyOpts struct {
	SkipBearerTokenCheckPaths []string
}

type ServicePolicyOpts struct {
	*ServiceBlockedIpsOpts
	*ServiceReqOriginOpts
	*ServiceReqAuthPolicyOpts
}

type ServiceValidatorOpts struct {
	RequiredBodyFields []string
	RequiredHeaders    []string
	RestrictedHeaders  []string
	AllowedHeaders     []string
}

type ServiceRedirectOpts struct {
	RouteLevelRedirection map[reqPath]redirectPath
	ProxyReqTimeout       *time.Duration
}

type ServiceVersionOpts struct {
	AvialableVersions []string
	DefaultVersion    string
}

type ServiceDepricationOpts struct {
	DeprecatedUrls    []reqPath
	DeprecatedHeaders []string
	ObsoleteUrls      []reqPath
}

type ServiceConfig struct {
	ServiceName        string
	ServiceUrl         string
	Timeout            *time.Duration
	RateLimitOpts      *ServiceRateLimitOpts
	ValidatorOpts      *ServiceValidatorOpts
	RedirectOpts       *ServiceRedirectOpts
	UrlDepricationOpts *ServiceDepricationOpts
	PolicyOpts         *ServicePolicyOpts
	VersionOpts        *ServiceVersionOpts
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
	ReqSizeLimit             int
	Services                 []ServiceConfig
	InternalOnlyServices     []string
	InternalOnlyServicesUrls []string
	BlockedIps               []string
	HealthCheckInterval      time.Duration
}

type ConfigLoader struct{}

func (ml *ConfigLoader) Load() (Config, error) {
	return Config{
		Port:                     5000,
		Timeout:                  10 * time.Second,
		RateLimit:                30,
		ReqSizeLimit:             5000,
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
