package ratelimit

type RateLimitOpts struct {
	GlobalLimits  int
	ServiceLimits int
	RouteLimits   int
}

type RateLimiter interface {
	Limit(RateLimitOpts) error
}
