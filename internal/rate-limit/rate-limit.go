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

type serviceTokenBucket struct {
	*tokenBucket
	urlTokenBucketMap map[string]*tokenBucket
	createdAt         time.Time
}

type serviceBucketMap map[string]*serviceTokenBucket

type buckets struct {
	globalTB *tokenBucket
	serviceBucketMap
	createdAt time.Time
}

type ipTBucketMap map[string]*buckets

type reqRateLimiter struct {
	globalRateLimitPerMin float64
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
		tb = &buckets{globalTB: gtb, serviceBucketMap: sbm, createdAt: time.Now()}
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

func (rrl *reqRateLimiter) Clean() {
	itbm := rrl.ipTBucketMap
	CLEANUP_TIME_MINUTES := 5.0

	for _, tb := range itbm {
		if tb == nil {
			continue
		}

		diff := time.Since(tb.createdAt).Minutes()

		if diff > CLEANUP_TIME_MINUTES {
			tb.globalTB = nil
		}

		for _, sbm := range tb.serviceBucketMap {

			diff := time.Since(sbm.createdAt).Minutes()

			if diff > CLEANUP_TIME_MINUTES {
				tb.globalTB = nil
			}

		}

	}
}

func NewReqRateLimiter(ctx context.Context, globalRateLimitPerMinute float64, services []config.ServiceConfig) *reqRateLimiter {

	rrl := reqRateLimiter{
		ctx:                   ctx,
		globalRateLimitPerMin: globalRateLimitPerMinute,
		serviceConfigs:        services,
		ipTBucketMap:          make(ipTBucketMap),
	}

	go func() {
		ticker := time.NewTicker(5 * time.Minute)

		for {
			select {
			case <-ticker.C:
			case <-ctx.Done():
				return
			}
		}

	}()

	return &rrl
}

// type servcieRefillData struct {
// 	refillTime *int
// 	url        map[string]int
// }

// type serviceRefillDataMap map[string]servcieRefillData
// type bucketsRefillData struct {
// 	globaRefillTime int
// 	services        serviceRefillDataMap
// }

func getBuckets(ctx context.Context, globalRateLimit float64, services []config.ServiceConfig) (*tokenBucket, serviceBucketMap) {

	cap, _, tokenCountToFill, rRate := calculateBucketData(globalRateLimit)
	gCtx, cancel := context.WithCancel(ctx)
	globalTb := newTokenbucket(gCtx, cap, tokenCountToFill, rRate, cancel)
	sbm := make(serviceBucketMap)

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

		sbm[service.ServiceName] = &serviceTokenBucket{
			tokenBucket:       stb,
			urlTokenBucketMap: utbm,
			createdAt:         time.Now(),
		}
		// brd.services[service.ServiceName] = srd

	}

	go func() {

		ticker := time.NewTicker(5 * time.Minute)

		destroyTBucket := func(tb *tokenBucket) {
			tb.destory()
		}

		destoryServiceTBucket := func(stb *serviceTokenBucket) {
			stb.destory()
			for _, utb := range stb.urlTokenBucketMap {
				destroyTBucket(utb)
			}
		}

		enoughIdleTimePassed := func(t time.Time) bool {
			min := time.Since(t).Minutes()
			if min < 5 {
				return false
			}
			return true
		}

		cleanupBuckets := func() {

			if len(globalTb.tokens) == globalTb.capacity && enoughIdleTimePassed(globalTb.lastUsed) {

				globalTb.destory()
				for _, stb := range sbm {
					destoryServiceTBucket(stb)
				}

				return
			}

			for _, stb := range sbm {
				if len(stb.tokens) == stb.capacity && enoughIdleTimePassed(stb.lastUsed) {
					destoryServiceTBucket(stb)
				}

				for k, utb := range stb.urlTokenBucketMap {
					if len(utb.tokens) == utb.capacity && enoughIdleTimePassed(utb.lastUsed) {
						delete(stb.urlTokenBucketMap, k)
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
