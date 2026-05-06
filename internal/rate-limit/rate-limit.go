package ratelimit

import "gateway/internal/config"

type RateLimitOpts struct {
	GlobalRouteLimits int
	config.ServiceRateLimitOpts
}

type RateLimiter interface {
	Limit(RateLimitOpts) error
}

type ReqRateLimiter struct{}

func (mrl ReqRateLimiter) Limit(RateLimitOpts) error {
	return nil
}
