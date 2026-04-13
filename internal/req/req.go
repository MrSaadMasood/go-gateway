package req

import (
	"gateway/internal/req/router"
	servicestore "gateway/internal/store"
)

type ReqHandler interface {
	Handle(servicestore.Storer) router.Router
}
