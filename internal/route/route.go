package route

import (
	"gateway/internal/config"
	"net/http"
)

type Router interface {
	Route(r *http.Request, rc *config.ServiceReqRoutingOpts) (string, error)
}

type ReqRouter struct{}

func NewReqRouter() *ReqRouter {
	return &ReqRouter{}
}

func (rr *ReqRouter) Route(r *http.Request, rro *config.ServiceReqRoutingOpts) (string, error) {
	if rro == nil {

	}

}
