package ratelimit

import (
	"context"
	"math"
	"sync"
	"time"
)

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
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				tb.fill()
			case <-ctx.Done():
				return
			}
		}
	}()

	return &tb

}

type serviceTB struct {
	tBucket           *tBucket
	urlTokenBucketMap map[string]*tBucket
	rwmu              sync.RWMutex
}

func (stb *serviceTB) setUrlTb(url string, tb *tBucket) {
	if tb == nil {
		return
	}

	stb.rwmu.Lock()
	defer stb.rwmu.Unlock()

	stb.urlTokenBucketMap[url] = tb
}

func (stb *serviceTB) getUrlTb(url string) *tBucket {
	stb.rwmu.RLock()
	defer stb.rwmu.RUnlock()

	tb, ok := stb.urlTokenBucketMap[url]
	if !ok || tb == nil {
		return nil
	}
	return tb

}

func (stb *serviceTB) untrackUrl(url string) {

	stb.rwmu.Lock()
	defer stb.rwmu.Unlock()

	utb, ok := stb.urlTokenBucketMap[url]
	if !ok || utb == nil {
		return
	}

	utb.stop()
	delete(stb.urlTokenBucketMap, url)
}

func (stb *serviceTB) consumeServiceToken() *token {
	stb.rwmu.RLock()
	defer stb.rwmu.RUnlock()

	if stb.tBucket == nil {
		return nil
	}
	return stb.tBucket.getToken()
}

func (stb *serviceTB) consumeUrlToken(path string) *token {

	stb.rwmu.RLock()
	defer stb.rwmu.RUnlock()

	utb, shouldLimit := stb.urlTokenBucketMap[path]

	if !shouldLimit {
		return &token{}
	}

	return utb.getToken()

}

type serviceBData struct {
	rwmu  *sync.RWMutex
	sbMap map[string]*serviceTB
}

func (sbm *serviceBData) setServiceTB(serviceName string, st *serviceTB) {
	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	sbm.sbMap[serviceName] = st
}

func (sbm *serviceBData) untrackAllSerices() {

	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	for _, stb := range sbm.sbMap {
		if stb.tBucket != nil {
			stb.tBucket.stop()
		}
		for _, utb := range stb.urlTokenBucketMap {
			if utb != nil {
				utb.stop()
			}
		}
	}

	sbm.sbMap = make(map[string]*serviceTB)

}

func (sbm *serviceBData) untrackService(serviceName string) {

	sbm.rwmu.Lock()
	defer sbm.rwmu.Unlock()

	stb, ok := sbm.sbMap[serviceName]
	if !ok || stb == nil {
		return
	}

	stb.tBucket.stop()
	for _, tb := range stb.urlTokenBucketMap {
		if tb != nil {
			tb.stop()
		}
	}

	delete(sbm.sbMap, serviceName)
}

type buckets struct {
	globalTB *tBucket
	*serviceBData
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

	sb, ok := b.serviceBData.sbMap[serviceName]
	if !ok || sb.tBucket == nil {
		return nil
	}

	return sb
}
