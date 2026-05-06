package proxy

import (
	"gateway/internal/config"
	"net/http"
)

type Proxier interface {
	Proxy(r *http.Request, ro config.ServiceRedirectOpts) (http.Response, error)
}

type ReqProxy struct{}

func (ReqProxy) Proxy(r *http.Request, ro config.ServiceRedirectOpts) (http.Response, error) {
	return http.Response{}, nil
}
