package ratelimit

import "gateway/internal/config"

type RateLimitOpts struct {
	GlobalRouteLimits int
	config.ServiceRateLimitOpts
}

type RateLimiter interface {
	Limit(RateLimitOpts) error
}
