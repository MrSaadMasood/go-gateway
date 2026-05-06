package gateway

type Gatewayer interface {
	Start() error
	Stop()
}

type Gateway struct {
}

func NewGateway() Gateway {
	return Gateway{}
}
