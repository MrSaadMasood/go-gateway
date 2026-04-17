package policy

type PolicyEnforcerOptions struct {
	ConcurrencyLimit        int
	Priority                int
	ExecutionContraintFuncs []func() error
}
type PolicyEnforcer interface {
	Enforce(PolicyEnforcerOptions)
}
