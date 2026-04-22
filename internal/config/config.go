package config

import (
	"net/http"
	"net/url"
	"strings"
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
	Timeout            time.Time
	RateLimitOpts      *ServiceRateLimitOpts
	ValidatorOpts      *ServiceValidatorOpts
	RedirectOpts       *ServiceRedirectOpts
	UrlDepricationOpts *ServiceDepricationOpts
	PolicyOpts         *ServicePolicyOpts
	VersionOpts        *ServiceVersionOpts
}

func (sc *ServiceConfig) HandleRequest(r *http.Request) {
	sc.checkDeprecations(r.URL.Path)
	sc.versionValidated()

}

func (sc *ServiceConfig) checkDeprecations(path string) bool {

}

func (sc *ServiceConfig) versionValidated(q url.Values, h http.Header, path string) bool {

	version := sc.VersionOpts.DefaultVersion
	versionFromQuery := q.Get("version")
	versionFromHeader := h.Get("version")
	pathSplit := strings.Split(path, "/")

	if versionFromQuery != "" {
		version = versionFromQuery
	}
	if versionFromHeader != "" {
		version = versionFromHeader
	}
	if len(pathSplit) > 1 {
		version = pathSplit[1]
	}

}

func (sc *ServiceConfig) enforcePolicies(r *http.Request) error {

}

func (sc *ServiceConfig) validateRequest(r *http.Request) error {

}

func (sc *ServiceConfig) rateLimit(r *http.Request) error {

}

func (sc *ServiceConfig) proxy(r *http.Request) error {

}

type Config struct {
	Port                     int
	Timeout                  time.Time
	RateLimit                int
	ReqSizeLimit             int
	Services                 []ServiceConfig
	InternalOnlyServices     []string
	InternalOnlyServicesUrls []string
	HealthCheckInterval      time.Time
	BlockedIps               []string
}

type Loader interface {
	Load() (Config, error)
}
