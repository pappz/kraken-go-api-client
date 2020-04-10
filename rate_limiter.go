package krakenapi

import (
	"log"
	"net/url"
	"sync"
	"time"
)

var (
	publicRule = rule{
		1,
		1 * time.Second,
	}

	privateStarterRule = rule{
		15,
		3 * time.Second,
	}

	privateIntermediateRule = rule{
		20,
		2 * time.Second,
	}

	privateProRule = rule{
		20,
		1 * time.Second,
	}
)

type rule struct {
	limit     uint
	reduction time.Duration
}

type call struct {
	api     *KrakenApi
	reqURL  string
	values  url.Values
	headers map[string]string
	typ     interface{}
	wg      sync.WaitGroup
	res     interface{}
	err     error
}

func newCall(api *KrakenApi, reqURL string, values url.Values, headers map[string]string, typ interface{}) *call {
	c := &call{
		api:     api,
		reqURL:  reqURL,
		values:  values,
		headers: headers,
		typ:     typ,
	}
	c.lock()
	return c
}

func (c *call) lock() {
	c.wg.Add(1)
}

func (c *call) done(res interface{}, err error) {
	c.res = res
	c.err = err
	c.wg.Done()
}

func (c *call) waitFor() (interface{}, error) {
	c.wg.Wait()
	return c.res, c.err
}

type rateLimiter struct {
	requestBucket uint
	rule          rule
	callList      chan *call
	mux           sync.Mutex
	lastDecrease  time.Time
	waitTime      time.Duration
}

func NewPublicRateLimit() *rateLimiter {
	rl := &rateLimiter{
		requestBucket: 0,
		rule:          publicRule,
		callList:      make(chan *call, 100),
		lastDecrease:  time.Now(),
		waitTime:      publicRule.reduction + (200 * time.Millisecond),
	}

	go rl.startTimer()
	go rl.startSender()
	return rl
}

func (r *rateLimiter) limitedRequest(api *KrakenApi, reqURL string, values url.Values, headers map[string]string, typ interface{}) (interface{}, error) {
	call := newCall(api, reqURL, values, headers, typ)
	r.callList <- call
	resp, err := call.waitFor()
	return resp, err
}

func (r *rateLimiter) startSender() {
	for {
		c := <-r.callList
		r.waitForChannel()
		r.increaseBucket()
		log.Printf("send out request: %s", time.Now())
		go r.doRequest(c)
	}
}

func (r *rateLimiter) waitForChannel() {
	if r.getBucketSize() < r.rule.limit {
		return
	}

	diff := r.waitTime - time.Since(r.lastDecrease)
	log.Printf("wait :%s", diff)
	time.Sleep(diff)
}

func (r *rateLimiter) doRequest(c *call) {
	resp, err := c.api.doRequest(c.reqURL, c.values, c.headers, c.typ)
	c.done(resp, err)
}

func (r *rateLimiter) startTimer() {
	for r.lastDecrease = range time.Tick(1 * time.Second) {
		r.decreaseBucket()
	}
}

func (r *rateLimiter) decreaseBucket() {
	r.mux.Lock()
	if r.requestBucket > 0 {
		r.requestBucket -= 1
	}
	r.mux.Unlock()
}

func (r *rateLimiter) increaseBucket() {
	r.mux.Lock()
	r.requestBucket += 1
	r.mux.Unlock()
}

func (r *rateLimiter) getBucketSize() uint {
	r.mux.Lock()
	defer r.mux.Unlock()
	return r.requestBucket
}
