package validate

import (
	"errors"
	"fmt"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net"
	"net/http"
	"slices"
	"strings"

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
}

type reqValidator struct {
	GlobalBlockedIps     []string
	GlobalAllowedOrigins []string
	GlobalReqSizeLimit   int
	ServiceConfigMap     config.ServiceConfigMap
}

func NewReqValidator(globalBlockedIps, globalAllowedOrigins []string, globalReqSizeLimit int, scm config.ServiceConfigMap) reqValidator {

	return reqValidator{
		GlobalBlockedIps:     globalBlockedIps,
		GlobalAllowedOrigins: globalAllowedOrigins,
		GlobalReqSizeLimit:   globalReqSizeLimit,
		ServiceConfigMap:     scm,
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
	if policyOpts == nil {
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

func (v reqValidator) validateCors(headers http.Header, sc *config.ServiceConfig) (http.HandlerFunc, error) {

	validatorOpts := sc.AuthOpts.ValidatorOpts
	if validatorOpts == nil {
		return func(w http.ResponseWriter, r *http.Request) {}, nil
	}

	for _, rh := range sc.AuthOpts.ValidatorOpts.RequiredHeaders {
		_, exists := headers[rh]
		if !exists {
			return nil, v.fail(http.StatusBadRequest, fmt.Errorf("missing header: %s", rh))
		}
	}

	for _, rh := range sc.AuthOpts.ValidatorOpts.RestrictedHeaders {
		_, exists := headers[rh]
		if exists {
			return nil, v.fail(http.StatusBadRequest, fmt.Errorf("header not allowed: %s", rh))
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
	if policyOpts == nil || !policyOpts.ShouldVerifyBearerToken || slices.Contains(policyOpts.SkipBearerTokenCheckPaths, path) {
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

func (v reqValidator) handleDeprecation() error {
	return nil
}

func (v reqValidator) validateVersion() error {
	return nil
}

func (v reqValidator) Validate(r *http.Request, w http.ResponseWriter, serviceName string) error {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return v.fail(http.StatusBadRequest, errors.New("failed to get the ip address of the request"))
	}

	s, ok := v.ServiceConfigMap[serviceName]
	if !ok {
		return v.fail(http.StatusBadRequest, errors.New("service not supported"))
	}

	err = v.validateIp(host, r.URL.Path, &s)
	if err != nil {
		return err
	}

	handlerFunc, err := v.validateCors(r.Header, &s)
	if err != nil {
		return err
	}

	wrw := NewWrappedResponseWriter(w)
	handlerFunc(wrw, r)
	if wrw.IsWritten() {
		return customerrors.ResponseAlreadySentErr
	}

	err = v.validateBearerToken(r.Header.Get("Authorization"), r.URL.Path, &s)
	if err != nil {
		return err
	}

	err = v.handleDeprecation()
	if err != nil {
		return err
	}

	err = v.validateVersion()
	if err != nil {
		return err
	}

	return nil
}
