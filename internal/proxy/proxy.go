package proxy

import (
	"context"
	"gateway/internal/config"
	"net/http"
	"net/url"
)

type Proxier interface {
	Proxy(ctx context.Context, method string, body *[]byte, h http.Header, url *url.URL, ro config.ServiceRedirectOpts) (http.Response, error)
}

type ReqProxy struct{}

func (ReqProxy) Proxy(ctx context.Context, method string, body *[]byte, h http.Header, url *url.URL, ro config.ServiceRedirectOpts) (http.Response, error) {
	return http.Response{}, nil
}
