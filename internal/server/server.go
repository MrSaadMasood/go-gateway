package server

import "gateway/internal/req"

type Server interface {
	Listen(req.ReqHandler)
	ShutDown()
}
