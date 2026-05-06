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

type ReqServiceAccessController struct{}

func (msc ReqServiceAccessController) Control(AccessControllerOpts) error {
	return nil
}
