package krakenapi

import (
	"log"
	"net/url"
	"sync"
	"time"
)

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
	sentTime  time.Time
	callList  chan *call
	extraWait bool
}

func NewPublicRateLimit() *rateLimiter {
	rl := &rateLimiter{
		callList: make(chan *call, 100),
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

func (r *rateLimiter) startSender() {

	for i := 0; ; i++ {
		c := <-r.callList

		if r.extraWait {
			time.Sleep(time.Second * 10)
			r.extraWait = false
		}

		r.sentTime = time.Now()
		go r.doRequest(c)
	}
}

func (r *rateLimiter) channelIsFree() bool {
	if r.extraWait {

		return false

	}
	return time.Since(r.sentTime) > time.Second
}

func (r *rateLimiter) waitForChannel() {
	diff := time.Second - time.Since(r.sentTime)
	if r.extraWait {
		diff = time.Second * 5
		defer r.cleanExtraWait()
	} else {
		diff = time.Millisecond * 0
	}

	log.Printf("wait: %s", diff)
	time.Sleep(diff)
}

func (r *rateLimiter) doRequest(c *call) {
	resp, err := c.api.doRequest(c.reqURL, c.values, c.headers, c.typ)
	if err != nil {
		r.extraWait = true
	}
	c.done(resp, err)
}

func (r *rateLimiter) cleanExtraWait() {
	r.extraWait = false
}
