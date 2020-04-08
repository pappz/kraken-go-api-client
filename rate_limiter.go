package krakenapi

import (
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
	calls     uint
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
	sentTime          time.Time
	requestsInNetwork uint
	rule              rule
	callList          chan *call
}

func NewPublicRateLimit() *rateLimiter {
	rl := &rateLimiter{
		sentTime:          time.Now(),
		requestsInNetwork: 0,
		rule:              publicRule,
		callList:          make(chan *call, 100),
	}

	go rl.startSender()
	return rl
}

func (r *rateLimiter) limitedRequest(api *KrakenApi, reqURL string, values url.Values, headers map[string]string, typ interface{}) (interface{}, error) {
	call := newCall(api, reqURL, values, headers, typ)
	r.callList <- call
	resp, err := call.waitFor()
	return resp, err
}

func (r *rateLimiter) acquire() {
	elapsedSeconds := uint(time.Since(r.sentTime).Seconds())
	decrease := elapsedSeconds * r.rule.calls
	r.requestsInNetwork -= decrease

	if r.requestsInNetwork < r.rule.calls {
		r.sentTime = time.Now()
		r.requestsInNetwork++
		return
	}

}

func (r *rateLimiter) startSender() {

	for {
		c := <-r.callList

		r.calculateFreeSlots()

		if !r.channelIsFree() {
			time.Sleep(1 * time.Second)
		}

		r.sentTime = time.Now()
		r.requestsInNetwork++
		go r.doRequest(c)
	}
}

func (r *rateLimiter) channelIsFree() bool {
	return r.requestsInNetwork < r.rule.calls
}

func (r *rateLimiter) calculateFreeSlots() {
	elapsedSeconds := uint(time.Since(r.sentTime).Seconds())
	decrease := elapsedSeconds * r.rule.calls
	r.requestsInNetwork -= decrease
}

func (r *rateLimiter) doRequest(c *call) {
	resp, err := c.api.doRequest(c.reqURL, c.values, c.headers, c.typ)
	c.done(resp, err)
}
