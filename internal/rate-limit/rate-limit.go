package ratelimit

import (
	"context"
	"errors"
	"gateway/internal/config"
	customerrors "gateway/internal/custom-errors"
	"gateway/internal/enums"
	"math"
	"net/http"
	"sync"
	"time"
)

var globalRateLimitErr = errors.New("global rate limits exceeded")
var serviceLevelRateLimitErr = errors.New("service level rate limits exceeded")
var routeLevelRateLimitErr = errors.New("route level rate limit exceeded")

type token struct{}

type tokenBucket struct {
	tokens           []token
	capacity         int
	rw               *sync.RWMutex
	tokenCountToFill int
	cancelCtx        context.CancelFunc
	lastUsed         time.Time
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
	tb.lastUsed = time.Now()
	return &t

}

func (tb *tokenBucket) fill() {

	tb.rw.Lock()
	defer tb.rw.Unlock()

	spaceAvailable := tb.capacity - len(tb.tokens)
	if spaceAvailable <= 0 {
		return
	}

	minTokensToFill := int(math.Min(float64(spaceAvailable), float64(tb.tokenCountToFill)))

	tb.tokens = append(tb.tokens, make([]token, minTokensToFill)...)
}

func (tb *tokenBucket) destory() {
	tb.rw.Lock()
	defer tb.rw.Unlock()

	tb.cancelCtx()
	tb = nil
}

func (tb *tokenBucket) isFull() bool {
	tb.rw.RLock()
	defer tb.rw.RUnlock()

	if len(tb.tokens) >= tb.capacity {
		return true
	}

	return false
}

func (tb *tokenBucket) isSittingIdle() bool {
	tb.rw.RLock()
	defer tb.rw.RUnlock()

	min := time.Since(tb.lastUsed).Minutes()
	if min < 5 {
		return false
	}
	return true
}

func newTokenbucket(ctx context.Context, capacity, tokenCountToFill int, refillRate time.Duration, cancelCtx context.CancelFunc) *tokenBucket {
	tb := tokenBucket{
		tokens:           make([]token, capacity),
		capacity:         capacity,
		rw:               &sync.RWMutex{},
		tokenCountToFill: tokenCountToFill,
		cancelCtx:        cancelCtx,
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

type serviceTB struct {
	*tokenBucket
	urlTBMap map[string]*tokenBucket
}

type serviceBMap map[string]*serviceTB

type buckets struct {
	globalTB *tokenBucket
	serviceBMap
}

type ipTBMap map[string]*buckets

type reqRateLimiter struct {
	globalRateLimitPerMin float64
	serviceConfigs        []config.ServiceConfig
	ctx                   context.Context
	ipTBucketMap          ipTBMap
	mu                    *sync.RWMutex
}

func (rrl *reqRateLimiter) Limit(serviceName, path, ip string) error {

	customErr := func(e error) error {
		return customerrors.NewReqFailedErr(http.StatusTooManyRequests, enums.ReqRateLimitFailed, e)
	}

	tb, ok := rrl.ipTBucketMap[ip]
	if !ok {
		gtb, sbm := getBuckets(rrl.ctx, rrl.globalRateLimitPerMin, rrl.serviceConfigs)
		tb = &buckets{globalTB: gtb, serviceBMap: sbm}
		rrl.ipTBucketMap[ip] = tb
	}

	gt := tb.globalTB
	if gt.getToken() == nil {
		return customErr(globalRateLimitErr)
	}

	sb, shouldLimit := tb.serviceBMap[serviceName]
	if !shouldLimit || sb.tokenBucket == nil {
		return nil
	}

	sbt := sb.tokenBucket
	if sbt.getToken() == nil {
		return customErr(serviceLevelRateLimitErr)
	}

	utb, shouldLimit := sb.urlTBMap[path]

	if !shouldLimit {
		return nil
	}

	if utb.getToken() == nil {
		return customErr(routeLevelRateLimitErr)
	}

	return nil
}

func NewReqRateLimiter(ctx context.Context, globalRateLimitPerMinute float64, services []config.ServiceConfig) *reqRateLimiter {

	rrl := reqRateLimiter{
		ctx:                   ctx,
		globalRateLimitPerMin: globalRateLimitPerMinute,
		serviceConfigs:        services,
		ipTBucketMap:          make(ipTBMap),
		mu:                    &sync.RWMutex{},
	}

	ticker := time.NewTicker(5 * time.Minute)

	go func() {

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}

	}()

	go func() {

		destoryServiceTBucket := func(stb *serviceTB) {
			stb.destory()
			for _, utb := range stb.urlTBMap {
				utb.destory()
			}
		}

		cleanupBuckets := func() {
			rrl.mu.Lock()
			defer rrl.mu.Unlock()

			for ip, b := range rrl.ipTBucketMap {
				if b.globalTB.isFull() && b.globalTB.isSittingIdle() {
					b.globalTB.destory()
					for _, stb := range b.serviceBMap {
						destoryServiceTBucket(stb)
					}

					delete(rrl.ipTBucketMap, ip)
					return
				}

				for service, stb := range b.serviceBMap {
					stbDestoryed := false

					if stb.isFull() && stb.isSittingIdle() {
						destoryServiceTBucket(stb)
						delete(b.serviceBMap, service)
						stbDestoryed = true
					}

					if stbDestoryed {
						continue
					}

					for url, utb := range stb.urlTBMap {
						if utb.isFull() && utb.isSittingIdle() {
							utb.destory()
							delete(stb.urlTBMap, url)
						}
					}
				}
			}
		}

		for {
			select {
			case <-ticker.C:
				cleanupBuckets()
			case <-ctx.Done():
				return
			}
		}

	}()

	return &rrl
}

func getBuckets(ctx context.Context, globalRateLimit float64, services []config.ServiceConfig) (*tokenBucket, serviceBMap) {

	cap, _, tokenCountToFill, rRate := calculateBucketData(globalRateLimit)
	gCtx, cancel := context.WithCancel(ctx)
	globalTb := newTokenbucket(gCtx, cap, tokenCountToFill, rRate, cancel)
	sbm := make(serviceBMap)

	for _, service := range services {

		var stb *tokenBucket = nil
		utbm := make(map[string]*tokenBucket)
		sCtx, cancel := context.WithCancel(gCtx)

		sro := service.RateLimitOpts
		if sro == nil {
			continue
		}

		srl := sro.RateLimit

		if srl != nil {
			cap, _, tokenCountToFill, rRate := calculateBucketData(*srl)
			stb = newTokenbucket(sCtx, cap, tokenCountToFill, rRate, cancel)
		}

		if sro.RouteLevelRateLimits != nil {
			for url, limit := range sro.RouteLevelRateLimits {
				cap, _, tokenCountToFill, rRate := calculateBucketData(limit)
				uCtx, cancel := context.WithCancel(sCtx)
				srtb := newTokenbucket(uCtx, cap, tokenCountToFill, rRate, cancel)
				utbm[url] = srtb
			}
		}

		sbm[service.ServiceName] = &serviceTB{
			tokenBucket: stb,
			urlTBMap:    utbm,
		}

	}

	return globalTb, sbm
}

func calculateBucketData(reqLimitPerMin float64) (int, int, int, time.Duration) {
	reqLimitPerMin = math.Min(4000, reqLimitPerMin)
	maxReqBurstAllowed := reqLimitPerMin * 2
	capacity := int(maxReqBurstAllowed)

	reqRefillTime := 60 / reqLimitPerMin
	reqProcessedPerSec := reqLimitPerMin / 60
	minRefillTime := math.Max(reqRefillTime, 1)

	if minRefillTime > 1 {
		reqProcessedPerSec = 1
	}

	completeRefillTime := maxReqBurstAllowed * reqRefillTime

	return capacity, int(reqProcessedPerSec), int(completeRefillTime), time.Duration(minRefillTime * float64(time.Second))
}
