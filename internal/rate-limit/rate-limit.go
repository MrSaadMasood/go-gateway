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
var FALLBACK_RATE_LIMIT = 3000.0

type token struct{}

type tBucket struct {
	rw        *sync.RWMutex
	cancelCtx context.CancelFunc

	tokens           []token
	capacity         int
	tokenCountToFill int
	lastUsed         time.Time
}

func (tb *tBucket) getToken() *token {

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

func (tb *tBucket) fill() {

	tb.rw.Lock()
	defer tb.rw.Unlock()

	spaceAvailable := tb.capacity - len(tb.tokens)
	if spaceAvailable <= 0 {
		return
	}

	minTokensToFill := int(math.Min(float64(spaceAvailable), float64(tb.tokenCountToFill)))

	tb.tokens = append(tb.tokens, make([]token, minTokensToFill)...)
}

func (tb *tBucket) isFull() bool {
	tb.rw.RLock()
	defer tb.rw.RUnlock()

	return len(tb.tokens) >= tb.capacity
}

func (tb *tBucket) stop() {
	tb.rw.Lock()
	defer tb.rw.Unlock()

	tb.cancelCtx()
	tb.tokens = make([]token, 0)
	tb.capacity = 0
	tb.tokenCountToFill = 0
	tb.lastUsed = time.Now()
}

func (tb *tBucket) isSittingIdle() bool {
	lastUserMinutes := time.Since(tb.lastUsed).Minutes()
	return lastUserMinutes >= 5 && tb.isFull()
}

func newTokenbucket(ctx context.Context, capacity, tokenCountToFill int, refillRate time.Duration, cancelCtx context.CancelFunc) *tBucket {
	tb := tBucket{
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
	tBucket           *tBucket
	urlTokenBucketMap map[string]*tBucket
	rwmu              sync.RWMutex
}

func (stb *serviceTB) untrackAllUrls() {

	stb.rwmu.Lock()
	defer stb.rwmu.Unlock()

	for _, utb := range stb.urlTokenBucketMap {
		utb.stop()
	}
	stb.urlTokenBucketMap = make(map[string]*tBucket)

}

func (stb *serviceTB) untrackUrl(url string) {

	stb.rwmu.Lock()
	defer stb.rwmu.Unlock()

	utb, ok := stb.urlTokenBucketMap[url]
	if !ok {
		return
	}

	utb.stop()
	delete(stb.urlTokenBucketMap, url)
}

func (stb *serviceTB) consumeServiceToken() *token {
	return stb.tBucket.getToken()
}

func (stb *serviceTB) consumeUrlToken(path string) *token {

	stb.rwmu.RLock()
	defer stb.rwmu.RUnlock()

	utb, shouldLimit := stb.urlTokenBucketMap[path]

	if !shouldLimit {
		return nil
	}

	return utb.getToken()

}

type serviceBucketsData struct {
	rwmu  *sync.RWMutex
	sbMap map[string]*serviceTB
}

func (sbm *serviceBucketsData) setServiceTB(serviceName string, st *serviceTB) {
	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	sbm.sbMap[serviceName] = st
}

func (sbm *serviceBucketsData) untrackAllSerices() {

	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	for _, stb := range sbm.sbMap {
		stb.tBucket.stop()
		stb.untrackAllUrls()
	}

	sbm.sbMap = make(map[string]*serviceTB)

}

func (sbm *serviceBucketsData) untrackService(serviceName string) {

	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	stb, ok := sbm.sbMap[serviceName]
	if !ok {
		return
	}

	stb.tBucket.stop()
	for url, _ := range stb.urlTokenBucketMap {
		stb.untrackUrl(url)
	}

	delete(sbm.sbMap, serviceName)
}

type buckets struct {
	globalTB *tBucket
	*serviceBucketsData
	rwmu sync.RWMutex
}

func (b *buckets) consumeGlobalToken() *token {
	if b.globalTB == nil {
		return nil
	}

	t := b.globalTB.getToken()
	return t
}

func (b *buckets) getServiceTb(serviceName string) *serviceTB {

	b.rwmu.RLock()
	defer b.rwmu.RUnlock()

	sb, ok := b.serviceBucketsData.sbMap[serviceName]
	if !ok || sb.tBucket == nil {
		return nil
	}

	return sb
}

type ipTBMap map[string]*buckets

type reqRateLimiter struct {
	globalRateLimitPerMin float64
	serviceConfigsMap     map[string]config.ServiceConfig
	ctx                   context.Context
	ipTBucketMap          ipTBucketMap
	rwmu                  sync.RWMutex
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

	gtb, sbm := rrl.getBuckets(ip, serviceName, path)
	if !b {
		b = &buckets{globalTB: gtb, serviceBucketsData: sbm, createdAt: time.Now(), rwmu: sync.RWMutex{}}
		rrl.ipTBucketMap[ip] = b
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
	b.serviceBucketsData.untrackAllSerices()

	delete(rrl.ipTBucketMap, ip)
}

func NewReqRateLimiter(ctx context.Context, globalRateLimitPerMinute float64, services []config.ServiceConfig) *reqRateLimiter {

	scm := make(map[string]config.ServiceConfig)

	for _, s := range services {
		scm[s.ServiceName] = s
	}

	rrl := reqRateLimiter{
		ctx:                   ctx,
		globalRateLimitPerMin: globalRateLimitPerMinute,
		serviceConfigsMap:     scm,
		ipTBucketMap:          make(ipTBucketMap),
		rwmu:                  sync.RWMutex{},
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

	go func() {

		cleanupBuckets := func() {
			for ip, b := range rrl.ipTBucketMap {

				if b.globalTB.isSittingIdle() {
					rrl.UntrackIp(ip)
					continue
				}

				for serviceName, stb := range b.serviceBucketsData.sbMap {
					if stb.tBucket.isSittingIdle() {
						b.serviceBucketsData.untrackService(serviceName)
						continue
					}

					for url, utb := range stb.urlTokenBucketMap {
						if utb.isSittingIdle() {
							stb.untrackUrl(url)
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

func (rrl *reqRateLimiter) setIpTBucket(ip string, b *buckets) {

	rrl.rwmu.Lock()
	defer rrl.rwmu.Unlock()

	rrl.ipTBucketMap[ip] = b
}

func (rrl *reqRateLimiter) getBuckets(ip, serviceName, path string) (*tBucket, *serviceBucketsData, error) {

	sc, ok := rrl.serviceConfigsMap[serviceName]
	if !ok {
		return nil, nil, errors.New("invalid service name provided for rate limiting")
	}

	cap, _, tokenCountToFill, rRate := calculateBucketData(rrl.globalRateLimitPerMin)
	gCtx, cancel := context.WithCancel(rrl.ctx)

	b := rrl.getIpTBucket(ip)
	var globalTB *tBucket
	var sbd *serviceBucketsData = nil
	defer func() {
		b := &buckets{globalTB: globalTB, serviceBucketsData: sbd, rwmu: sync.RWMutex{}}
		rrl.setIpTBucket(ip, b)
	}()

	if b == nil || b.globalTB == nil {
		globalTB = newTokenbucket(gCtx, cap, tokenCountToFill, rRate, cancel)
	} else {
		globalTB = b.globalTB
	}

	if b == nil || b.serviceBucketsData == nil {
		sbd = &serviceBucketsData{rwmu: &sync.RWMutex{}, sbMap: make(map[string]*serviceTB)}
	} else {
		sbd = b.serviceBucketsData
	}

	sro := sc.RateLimitOpts
	if sro == nil {
		return globalTB, nil, nil
	}

	sCtx, cancel := context.WithCancel(gCtx)
	srl := sro.RateLimit
	if srl == nil {
		srl = &FALLBACK_RATE_LIMIT
	}

	cap, _, tokenCountToFill, rRate = calculateBucketData(*srl)
	tb := newTokenbucket(sCtx, cap, tokenCountToFill, rRate, cancel)
	stbd := b.getServiceTb(serviceName)

	if stbd == nil {
		stbd = &serviceTB{
			tBucket:           tb,
			urlTokenBucketMap: make(map[string]*tBucket),
			rwmu:              sync.RWMutex{},
		}
	} else if stbd != nil && stbd.tBucket == nil {

		stbd = &serviceTB{
			tBucket:           tb,
			urlTokenBucketMap: make(map[string]*tBucket),
			rwmu:              sync.RWMutex{},
		}
	}

	if sro.RouteLevelRateLimits != nil {
		for url, limit := range sro.RouteLevelRateLimits {
			cap, _, tokenCountToFill, rRate := calculateBucketData(limit)
			uCtx, cancel := context.WithCancel(sCtx)
			srtb := newTokenbucket(uCtx, cap, tokenCountToFill, rRate, cancel)
			utbm[url] = srtb
		}
	}

	sbd.sbMap[sc.ServiceName] = &serviceTB{
		tBucket:           stbd,
		urlTokenBucketMap: utbm,
		rwmu:              sync.RWMutex{},
	}

	sbd.setServiceTB(serviceName, stbd)
	return globalTB, sbd, nil
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
