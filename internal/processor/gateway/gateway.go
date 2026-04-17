package gateway

import "net/http"

type Gateway interface {
	Start(port int, h http.Handler) error
	Stop()
}
