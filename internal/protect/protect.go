package protect

type ServiceOptions struct {
	ServiceName string
}

type ServiceProtector interface {
	Protect([]ServiceOptions) error
}
