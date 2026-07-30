package proxy

import (
	"bytes"
	"context"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net/http"
	"time"
)

type Proxier interface {
	Proxy(ctx context.Context, route string, method string, body *[]byte, h http.Header, proxyTimeout time.Duration) (http.Response, error)
}

type reqProxy struct {
	timeout time.Duration
}

func NewReqProxy(defaultTimeout time.Duration) *reqProxy {
	return &reqProxy{timeout: defaultTimeout}
}

func (r *reqProxy) Proxy(ctx context.Context, targetEndpoint string, method string, body *[]byte, headers http.Header, proxyTimeout time.Duration) (http.Response, error) {

	c := http.Client{Timeout: proxyTimeout}

	req, err := http.NewRequest(method, targetEndpoint, bytes.NewReader(*body))
	for h, v := range headers {
		for _, value := range v {
			req.Header.Add(h, value)
		}
	}

	if err != nil {
		return http.Response{}, customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqProxyFailed, err)
	}

	res, err := c.Do(req)
	if err != nil {
		return http.Response{}, customerrors.NewReqFailedErr(http.StatusBadRequest, enums.ReqProxyFailed, err)
	}

	return *res, nil
}
