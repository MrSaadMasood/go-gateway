package gateway

type Gatewayer interface {
	Start(port int) error
	Stop()
}
