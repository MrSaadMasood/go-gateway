package proxy

type Request interface {
	Write([]byte) (int, error)
	WriteHeader(statusCode int)
}

type ServiceProxier interface {
	Proxy(Request) error
}
