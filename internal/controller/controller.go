package controller

type AccessControllerOpts struct {
	RestrictedServices []string
	RestrictedPaths    []string
	ServiceName        string
	ReqPath            string
}

type ServiceAccessController interface {
	Control(AccessControllerOpts) error
}
