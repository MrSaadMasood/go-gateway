package validate

import (
	"errors"
	"fmt"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/rs/cors"
)

type wrappedResponseWriter struct {
	http.ResponseWriter
	written bool
}

func NewWrappedResponseWriter(w http.ResponseWriter) *wrappedResponseWriter {
	return &wrappedResponseWriter{w, false}
}

func (wrw *wrappedResponseWriter) IsWritten() bool {
	return wrw.written
}

func (wrw *wrappedResponseWriter) Write(b []byte) (int, error) {
	wrw.written = true
	return wrw.ResponseWriter.Write(b)
}

func (wrw *wrappedResponseWriter) Header() http.Header {
	return wrw.Header()
}

func (wrw *wrappedResponseWriter) WriteHeader(statusCode int) {
	wrw.written = true
	wrw.ResponseWriter.WriteHeader(statusCode)
}

type Validator interface {
	Validate(req *http.Request, w http.ResponseWriter, serviceName string) error
	ValidateReqSize(bodySizeInBytes int) error
}

type reqValidator struct {
	GlobalBlockedIps          []string
	GlobalAllowedOrigins      []string
	GlobalReqSizeLimitInBytes int
	ServiceConfigMap          config.ServiceConfigMap
}

func NewReqValidator(globalBlockedIps, globalAllowedOrigins []string, globalReqSizeLimit int, scm config.ServiceConfigMap) reqValidator {

	return reqValidator{
		GlobalBlockedIps:          globalBlockedIps,
		GlobalAllowedOrigins:      globalAllowedOrigins,
		GlobalReqSizeLimitInBytes: globalReqSizeLimit,
		ServiceConfigMap:          scm,
	}
}

func (reqValidator) fail(code int, err error) error {
	return customerrors.NewReqFailedErr(code, enums.ReqValidationFailed, err)
}

func (v reqValidator) validateIp(ip string, path string, sc *config.ServiceConfig) error {

	if slices.Contains(v.GlobalBlockedIps, ip) {
		return v.fail(http.StatusForbidden, errors.New("Forbidden"))
	}

	policyOpts := sc.AuthOpts.PolicyOpts
	if policyOpts == nil || policyOpts.ServiceBlockedIpsOpts == nil {
		return nil
	}

	if slices.Contains(policyOpts.BlockedIps, ip) {
		return v.fail(http.StatusForbidden, errors.New("Forbidden"))
	}

	routeWithBlockedIps, ok := policyOpts.RouteLevelBlockedIps[path]
	if ok && slices.Contains(routeWithBlockedIps, ip) {
		return v.fail(http.StatusForbidden, errors.New("Forbidden"))
	}

	return nil
}

func (v reqValidator) addDeprecationHeaders(resHeaders http.Header, msg string) {
	resHeaders.Add("Deprecation", "@"+strconv.Itoa(int(time.Now().Unix())))
	resHeaders.Add("Warning", msg)
}

func (v reqValidator) validateCors(reqHeaders http.Header, resHeaders http.Header, sc *config.ServiceConfig) (http.HandlerFunc, error) {

	validatorOpts := sc.AuthOpts.ValidatorOpts
	if validatorOpts == nil {
		return func(w http.ResponseWriter, r *http.Request) {}, nil
	}

	for _, rh := range sc.AuthOpts.ValidatorOpts.RequiredHeaders {
		_, exists := reqHeaders[rh]
		if !exists {
			return nil, v.fail(http.StatusBadRequest, fmt.Errorf("missing header: %s", rh))
		}
	}

	for _, rh := range sc.AuthOpts.ValidatorOpts.RestrictedHeaders {
		_, exists := reqHeaders[rh]
		if exists {
			return nil, v.fail(http.StatusBadRequest, fmt.Errorf("header not allowed: %s", rh))
		}
	}

	deprecationOpts := sc.AuthOpts.DeprecationOpts
	if deprecationOpts != nil {
		for _, dh := range deprecationOpts.DeprecatedHeaders {
			_, exists := reqHeaders[dh]
			if exists {
				v.addDeprecationHeaders(resHeaders, fmt.Sprintf("299 - The header: %s is deprecated. Please refer to the api documentation for latest supported headers", dh))
			}
		}
	}

	policyOpts := sc.AuthOpts.PolicyOpts
	if policyOpts == nil {
		return func(w http.ResponseWriter, r *http.Request) {}, nil
	}

	corsPolicy := cors.New(cors.Options{
		AllowedOrigins: append([]string{}, policyOpts.AllowedOrigins...),
		AllowedHeaders: append([]string{}, validatorOpts.AllowedHeaders...),
	})

	return corsPolicy.HandlerFunc, nil
}

func (v reqValidator) validateBearerToken(token string, path string, sc *config.ServiceConfig) error {
	policyOpts := sc.AuthOpts.PolicyOpts
	if policyOpts == nil || policyOpts.ServiceBearerTokenPolicyOpts == nil {
		return nil
	}
	if !policyOpts.ShouldVerifyBearerToken || slices.Contains(policyOpts.SkipBearerTokenCheckPaths, path) {
		return nil
	}

	if token == "" {
		return v.fail(http.StatusUnauthorized, errors.New("no token provided"))
	}

	if !strings.HasPrefix(token, "Bearer ") {
		return v.fail(http.StatusUnauthorized, errors.New("invalid token format"))
	}

	return nil
}

func (v reqValidator) handleDeprecation(path string, resHeaders http.Header, sc *config.ServiceConfig) error {
	deprecationOpts := sc.AuthOpts.DeprecationOpts
	if deprecationOpts == nil {
		return nil
	}

	for _, oUrl := range sc.AuthOpts.DeprecationOpts.ObsoleteUrls {
		if strings.Contains(string(oUrl), path) {
			return v.fail(http.StatusNotFound, errors.New("path not found"))
		}
	}

	for _, dUrl := range sc.AuthOpts.DeprecationOpts.DeprecatedUrls {
		if strings.Contains(string(dUrl), path) {
			v.addDeprecationHeaders(resHeaders, "299 - The url is deprecated. Please refer to the api documentation for latest supported paths")
		}
	}

	return nil

}

func (v reqValidator) validateVersion(serviceVersion string, sc *config.ServiceConfig) error {
	versionOpts := sc.AuthOpts.VersionOpts
	if versionOpts == nil {
		return nil
	}

	for _, version := range sc.AuthOpts.VersionOpts.AvialableVersions {
		if strings.Contains(version, serviceVersion) {
			return nil
		}
	}

	return v.fail(http.StatusNotFound, errors.New("the api version is not supported"))

}

func (v reqValidator) ValidateReqSize(bodySizeInBytes int) error {
	if bodySizeInBytes > v.GlobalReqSizeLimitInBytes {
		return v.fail(http.StatusUnprocessableEntity, errors.New("req body too large to process"))
	}
	return nil
}

func (v reqValidator) Validate(w http.ResponseWriter, r *http.Request, serviceName string) error {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return v.fail(http.StatusBadRequest, errors.New("failed to get the ip address of the request"))
	}

	s, ok := v.ServiceConfigMap[serviceName]
	if !ok {
		return v.fail(http.StatusBadRequest, errors.New("service not supported"))
	}

	splitted := strings.SplitN(r.URL.Path, "/", 4)
	if len(splitted) < 3 {
		return v.fail(http.StatusBadRequest, errors.New("req path not valid "))
	}

	m, err := regexp.Match(`^v\d+$`, []byte(splitted[2]))
	if err != nil || m == false {
		return v.fail(http.StatusBadRequest, errors.New("version verification failed"))
	}

	reqVersion := splitted[2]
	var path string = "/"
	if len(splitted) == 4 {
		path = splitted[3]
	}

	err = v.validateIp(host, path, &s)
	if err != nil {
		return err
	}

	handlerFunc, err := v.validateCors(r.Header, w.Header(), &s)
	if err != nil {
		return err
	}

	wrw := NewWrappedResponseWriter(w)
	handlerFunc(wrw, r)
	if wrw.IsWritten() {
		return customerrors.ResponseAlreadySentErr
	}

	err = v.validateBearerToken(r.Header.Get("Authorization"), path, &s)
	if err != nil {
		return err
	}

	err = v.handleDeprecation(path, w.Header(), &s)
	if err != nil {
		return err
	}

	err = v.validateVersion(reqVersion, &s)
	if err != nil {
		return err
	}

	return nil
}
