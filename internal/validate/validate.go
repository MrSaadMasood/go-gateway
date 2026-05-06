package validate

import "gateway/internal/config"

type ValidationOpts struct {
	GlobalBlockedIps   []string
	GlobalReqSizeLimit int
	ServiceOpts        config.ServiceConfig
}

type Validator interface {
	Validate(ValidationOpts) error
}

type ReqValidator struct{}

func (ReqValidator) Validate(ValidationOpts) error {
	return nil
}
