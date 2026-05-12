package proxy

import (
	"gateway/internal/config"
	"io"
	"net/http"
	"net/url"
)

type Proxier interface {
	Proxy(method string, body io.ReadCloser, h http.Header, url *url.URL, ro config.ServiceRedirectOpts) (http.Response, error)
}

type ReqProxy struct{}

func (ReqProxy) Proxy(method string, body io.ReadCloser, h http.Header, url *url.URL, ro config.ServiceRedirectOpts) (http.Response, error) {
	return http.Response{}, nil
}
