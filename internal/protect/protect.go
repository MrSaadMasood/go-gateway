package protect

type ServiceOptions struct {
	ServiceName                string
	ShouldMitigateBackpressure bool
}

type ServiceProtector interface {
	Protect([]ServiceOptions) error
}
