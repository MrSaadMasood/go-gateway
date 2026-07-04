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
var FALLBACK_RATE_LIMIT = 2000.0

type RateLimiter interface {
	Limit(serviceName, path, ip string) error
}

type ipTBucketMap map[string]*buckets

type reqRateLimiter struct {
	globalRateLimitPerMin float64
	serviceConfigsMap     map[string]config.ServiceConfig
	ctx                   context.Context
	ipTBucketMap          ipTBucketMap
	rwmu                  sync.RWMutex
}

func NewReqRateLimiter(ctx context.Context, globalRateLimitPerMinute float64, scm config.ServiceConfigMap) *reqRateLimiter {

	rrl := reqRateLimiter{
		ctx:                   ctx,
		globalRateLimitPerMin: globalRateLimitPerMinute,
		serviceConfigsMap:     scm,
		ipTBucketMap:          make(ipTBucketMap),
		rwmu:                  sync.RWMutex{},
	}

	cleanupBuckets := func() {
		for ip, b := range rrl.ipTBucketMap {

			if b.globalTB != nil && b.globalTB.isSittingIdle() {
				rrl.UntrackIp(ip)
				continue
			}

			for serviceName, stb := range b.serviceBData.sbMap {
				if stb.tBucket != nil && stb.tBucket.isSittingIdle() {
					b.serviceBData.untrackService(serviceName)
					continue
				}

				for url, utb := range stb.urlTokenBucketMap {
					if utb != nil && utb.isSittingIdle() {
						stb.untrackUrl(url)
					}
				}
			}

		}
	}

	go func() {

		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

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

func (rrl *reqRateLimiter) getIpTBucket(ip string) *buckets {

	rrl.rwmu.RLock()
	defer rrl.rwmu.RUnlock()

	b, ok := rrl.ipTBucketMap[ip]
	if !ok {
		return nil
	}
	return b
}

func (rrl *reqRateLimiter) Limit(serviceName, path, ip string) error {

	customErr := func(e error) error {
		return customerrors.NewReqFailedErr(http.StatusTooManyRequests, enums.ReqRateLimitFailed, e)
	}

	b, err := rrl.getBuckets(ip, serviceName, path)
	if err != nil {
		return err
	}

	if b == nil {
		return errors.New("failed to get rate limits")
	}

	if b.consumeGlobalToken() == nil {
		return customErr(globalRateLimitErr)
	}

	sb := b.getServiceTb(serviceName)
	if sb == nil || sb.consumeServiceToken() == nil {
		return customErr(serviceLevelRateLimitErr)
	}

	if sb.consumeUrlToken(path) == nil {
		return customErr(routeLevelRateLimitErr)
	}

	return nil
}

func (rrl *reqRateLimiter) UntrackIp(ip string) {
	rrl.rwmu.Lock()
	defer rrl.rwmu.Unlock()

	b, ok := rrl.ipTBucketMap[ip]
	if !ok {
		return
	}

	b.globalTB.stop()
	b.serviceBData.untrackAllSerices()

	delete(rrl.ipTBucketMap, ip)
}

func (rrl *reqRateLimiter) setIpTBucket(ip string, b *buckets) {

	rrl.rwmu.Lock()
	defer rrl.rwmu.Unlock()

	rrl.ipTBucketMap[ip] = b
}

func (rrl *reqRateLimiter) getBuckets(ip, serviceName, path string) (*buckets, error) {

	sc, ok := rrl.serviceConfigsMap[serviceName]
	if !ok {
		return nil, errors.New("invalid service name provided for rate limiting")
	}

	gCtx, cancel := context.WithCancel(rrl.ctx)
	b := rrl.getIpTBucket(ip)
	var globalTB *tBucket = nil
	var sbd *serviceBData = nil

	if b == nil || b.globalTB == nil {
		cap, _, tokenCountToFill, rRate := calculateBucketData(rrl.globalRateLimitPerMin)
		globalTB = newTokenbucket(gCtx, cap, tokenCountToFill, rRate, cancel)
	} else {
		globalTB = b.globalTB
	}

	if b == nil || b.serviceBData == nil {
		sbd = &serviceBData{rwmu: &sync.RWMutex{}, sbMap: make(map[string]*serviceTB)}
	} else {
		sbd = b.serviceBData
	}

	if b == nil {
		b = &buckets{globalTB: globalTB, serviceBData: sbd, rwmu: sync.RWMutex{}}
	}

	var serviceRateLimit float64

	if sc.RateLimitOpts == nil || sc.RateLimitOpts.RateLimitPerMinute == nil {
		serviceRateLimit = FALLBACK_RATE_LIMIT
	} else {
		serviceRateLimit = *sc.RateLimitOpts.RateLimitPerMinute
	}

	sTBucket := b.getServiceTb(serviceName)

	if sTBucket == nil {
		sCtx, cancel := context.WithCancel(gCtx)
		cap, _, tokenCountToFill, rRate := calculateBucketData(serviceRateLimit)
		stb := newTokenbucket(sCtx, cap, tokenCountToFill, rRate, cancel)

		sTBucket = &serviceTB{
			tBucket:           stb,
			urlTokenBucketMap: make(map[string]*tBucket),
			rwmu:              sync.RWMutex{},
		}

	}

	if sTBucket.tBucket == nil {
		sCtx, cancel := context.WithCancel(gCtx)
		cap, _, tokenCountToFill, rRate := calculateBucketData(serviceRateLimit)
		stb := newTokenbucket(sCtx, cap, tokenCountToFill, rRate, cancel)
		sTBucket.tBucket = stb
	}

	utbm := sTBucket.urlTokenBucketMap
	if utbm == nil {
		utbm = make(map[string]*tBucket)
	}

	if sc.RateLimitOpts != nil && sc.RateLimitOpts.RouteLevelRateLimitsPerMinute != nil {
		for url, limit := range sc.RateLimitOpts.RouteLevelRateLimitsPerMinute {

			utb := sTBucket.getUrlTb(url)
			if utb == nil {
				sCtx, cancel := context.WithCancel(gCtx)
				uCtx, cancel := context.WithCancel(sCtx)
				cap, _, tokenCountToFill, rRate := calculateBucketData(limit)
				utb = newTokenbucket(uCtx, cap, tokenCountToFill, rRate, cancel)
				sTBucket.setUrlTb(url, utb)
			}
		}
	}

	sbd.setServiceTB(sc.ServiceName, sTBucket)
	rrl.setIpTBucket(ip, b)

	return b, nil
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
