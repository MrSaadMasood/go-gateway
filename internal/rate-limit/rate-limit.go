package ratelimit

import (
	"context"
	"errors"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"net/http"
	"sync"
	"time"
)

var globalRateLimitErr = errors.New("global rate limits exceeded")
var serviceLevelRateLimitErr = errors.New("service level rate limits exceeded")
var routeLevelRateLimitErr = errors.New("route level rate limit exceeded")

type token struct{}

type tokenBucket struct {
	tokens   []token
	capacity int
	rw       *sync.RWMutex
}

func (tb *tokenBucket) getToken() *token {

	tb.rw.Lock()
	defer tb.rw.Unlock()

	tokens := tb.tokens
	if len(tokens) == 0 {
		return nil
	}

	t, tokens := tokens[len(tokens)-1], tokens[:len(tokens)-1]
	tb.tokens = tokens
	return &t

}

func (tb *tokenBucket) fill() {

	tb.rw.Lock()
	defer tb.rw.Unlock()

	if len(tb.tokens) >= tb.capacity {
		return
	}

	tb.tokens = append(tb.tokens, token{})
}

func newTokenbucket(ctx context.Context, capacity int, refillRate time.Duration) *tokenBucket {
	tb := tokenBucket{
		tokens:   make([]token, capacity),
		capacity: capacity,
		rw:       &sync.RWMutex{},
	}

	go func() {
		ticker := time.NewTicker(refillRate)
		for {
			select {
			case <-ticker.C:
				tb.fill()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return &tb

}

type RateLimiter interface {
	Limit(serviceName, path, ip string) error
}

type serviceTokenBucket struct {
	*tokenBucket
	urlTokenBucketMap map[string]*tokenBucket
}

type serviceBucketMap map[string]*serviceTokenBucket

type buckets struct {
	globalTB *tokenBucket
	serviceBucketMap
}

type ipTBucketMap map[string]*buckets

type reqRateLimiter struct {
	globalRateLimitPerMin int
	serviceConfigs        []config.ServiceConfig
	ctx                   context.Context
	ipTBucketMap          ipTBucketMap
}

func (rrl *reqRateLimiter) Limit(serviceName, path, ip string) error {

	customErr := func(e error) error {
		return customerrors.NewReqFailedErr(http.StatusTooManyRequests, enums.ReqRateLimitFailed, e)
	}

	tb, ok := rrl.ipTBucketMap[ip]
	if !ok {
		gtb, sbm := getBuckets(rrl.ctx, rrl.globalRateLimitPerMin, rrl.serviceConfigs)
		tb = &buckets{globalTB: gtb, serviceBucketMap: sbm}
		rrl.ipTBucketMap[ip] = tb
	}

	gt := tb.globalTB
	if gt.getToken() == nil {
		return customErr(globalRateLimitErr)
	}

	sb, shouldLimit := tb.serviceBucketMap[serviceName]
	if !shouldLimit || sb.tokenBucket == nil {
		return nil
	}

	sbt := sb.tokenBucket
	if sbt.getToken() == nil {
		return customErr(serviceLevelRateLimitErr)
	}

	utb, shouldLimit := sb.urlTokenBucketMap[path]

	if !shouldLimit {
		return nil
	}

	if utb.getToken() == nil {
		return customErr(routeLevelRateLimitErr)
	}

	return nil
}

func NewReqRateLimiter(ctx context.Context, globalRateLimitPerMinute int, services []config.ServiceConfig) *reqRateLimiter {

	rrl := reqRateLimiter{
		ctx:                   ctx,
		globalRateLimitPerMin: globalRateLimitPerMinute,
		serviceConfigs:        services,
		ipTBucketMap:          make(ipTBucketMap),
	}

	return &rrl
}

func getBuckets(ctx context.Context, globalRateLimit int, services []config.ServiceConfig) (*tokenBucket, serviceBucketMap) {

	cap, rRate := calculateBucketData(globalRateLimit)
	globalTb := newTokenbucket(ctx, cap, rRate)
	sbm := make(serviceBucketMap)

	for _, service := range services {

		var stb *tokenBucket = nil
		utbm := make(map[string]*tokenBucket)

		sro := service.RateLimitOpts
		if sro == nil {
			continue
		}

		srl := sro.RateLimit
		if srl != nil {
			cap, rRate := calculateBucketData(*srl)
			stb = newTokenbucket(ctx, cap, rRate)
		}

		if sro.RouteLevelRateLimits != nil {
			for url, limit := range sro.RouteLevelRateLimits {
				cap, rRate := calculateBucketData(limit)
				srtb := newTokenbucket(ctx, cap, rRate)
				utbm[url] = srtb
			}
		}

		sbm[service.ServiceName] = &serviceTokenBucket{
			tokenBucket:       stb,
			urlTokenBucketMap: utbm,
		}

	}

	return globalTb, sbm
}

func calculateBucketData(reqLimit int) (int, time.Duration) {
	refillRate := float32(60) / float32(reqLimit)
	reqHandledPerMinute := float32(reqLimit) / 60
	maxBurstDurationInSeconds := 5
	capacity := int(reqHandledPerMinute * float32(maxBurstDurationInSeconds))
	refillInterval := time.Duration(refillRate * float32(time.Second))
	return capacity, refillInterval
}
