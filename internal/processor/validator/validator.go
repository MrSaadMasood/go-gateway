package validator

type ValidatorOpts struct {
	JobName string
	Payload map[string]any
}

type JobValidatorFunc func(ValidatorOpts) error
type JobValidatorMap map[string]JobValidatorFunc

type Validator interface {
	Validate(ValidatorOpts) error
}
