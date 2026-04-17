package controller

type ControlOpts struct {
	ShouldValidate        bool
	ShouldEnforcePolicies bool
	ShouldVersionControl  bool
}

type ServiceAccessController interface {
	Control(ControlOpts) error
}
