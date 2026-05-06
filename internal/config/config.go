package config

import (
	"time"
)

type reqPath string
type redirectPath string

type ServiceRateLimitOpts struct {
	RateLimit            *int
	RouteLevelRateLimits map[string]int
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
	Timeout            *time.Time
	RateLimitOpts      *ServiceRateLimitOpts
	ValidatorOpts      *ServiceValidatorOpts
	RedirectOpts       *ServiceRedirectOpts
	UrlDepricationOpts *ServiceDepricationOpts
	PolicyOpts         *ServicePolicyOpts
	VersionOpts        *ServiceVersionOpts
}

type Config struct {
	Port                     int
	Timeout                  time.Duration
	RateLimit                int
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
