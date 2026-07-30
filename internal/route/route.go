package route

import (
	"gateway/internal/common"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net/http"
	"regexp"
	"slices"
	"strings"
)

type MatcherFunc func(r *http.Request, rc config.RouteConfig) bool

type Router interface {
	Route(r *http.Request, rc *config.ServiceReqRoutingOpts) (string, error)
}

type ReqRouter struct{}

func NewReqRouter() *ReqRouter {
	return &ReqRouter{}
}

func (rr *ReqRouter) Route(r *http.Request, rro *config.ServiceReqRoutingOpts) (string, error) {

	path, _, err := common.RequestServiceExemptedPath(r.URL.Path)
	if err != nil {
		return "", customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqProxyFailed, err)
	}

	if rro != nil && rro.ReqRoutingConfigMap != nil {

		routeConfig, ok := rro.ReqRoutingConfigMap[config.ReqPath(path)]
		if !ok {
			return "", customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqProxyFailed, err)
		}

		for _, matchFunc := range rr.getRequestRedirectionMatchers() {
			matched := matchFunc(r, routeConfig)
			if matched {
				return routeConfig.RedirectPath, nil
			}
		}
	}

	return path, nil
}

func (rr *ReqRouter) headerMatcher(r *http.Request, rc config.RouteConfig) bool {

	for header, value := range rc.Headers {
		reqHeaders, ok := r.Header[header]

		if !ok {
			continue
		}

		matched := slices.Contains(reqHeaders, value)
		if matched {
			return true
		}
	}

	return false
}

func (rr *ReqRouter) headerRegexMatcher(r *http.Request, rc config.RouteConfig) bool {

	for header, regex := range rc.HeaderRegex {
		reqHeaders, ok := r.Header[header]
		if !ok {
			continue
		}

		for _, h := range reqHeaders {

			matched, err := regexp.MatchString(regex, h)
			if err == nil && matched == true {
				return true
			}

		}
	}

	return false
}

func (rr *ReqRouter) hostMatcher(r *http.Request, rc config.RouteConfig) bool {

	if strings.Contains(r.Host, rc.Host) {
		return true
	}

	return false
}

func (rr *ReqRouter) hostRegexMatcher(r *http.Request, rc config.RouteConfig) bool {

	matched, err := regexp.MatchString(rc.HostRegex, r.Host)
	if err == nil && matched == true {
		return true
	}

	return false
}

func (rr *ReqRouter) methodMatcher(r *http.Request, rc config.RouteConfig) bool {

	if slices.Contains(rc.Methods, r.Method) {
		return true
	}

	return false
}

func (rr *ReqRouter) queryMatcher(r *http.Request, rc config.RouteConfig) bool {

	for query, value := range rc.Query {

		qv := r.URL.Query().Get(query)
		if value == qv {
			return true
		}
	}

	return false
}

func (rr *ReqRouter) queryRegexMatcher(r *http.Request, rc config.RouteConfig) bool {

	for query, regex := range rc.QueryRegex {

		matched, err := regexp.MatchString(regex, r.URL.Query().Get(query))
		if err == nil && matched == true {
			return true
		}
	}

	return false
}

func (rr *ReqRouter) clientIpMatcher(r *http.Request, rc config.RouteConfig) bool {

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")

		if slices.Contains(ips, rc.ClientIp) {
			return true
		}
	}

	return false

}

func (rr ReqRouter) getRequestRedirectionMatchers() []MatcherFunc {
	return []MatcherFunc{
		rr.headerMatcher,
		rr.headerRegexMatcher,
		rr.hostMatcher,
		rr.hostRegexMatcher,
		rr.headerMatcher,
		rr.queryMatcher,
		rr.queryRegexMatcher,
		rr.clientIpMatcher,
	}
}
