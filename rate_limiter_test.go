package krakenapi

import (
	"log"
	"sync"
	"testing"
)

var (
	wg    sync.WaitGroup
	since = int64(1546470333801400000)
)

func sendRequest() {

	_, err := publicAPI.Trades("XXBTZCAD", since)
	if err != nil {
		log.Printf("%v", err)
	}
	wg.Done()
}

func TestNewPublicRateLimit(t *testing.T) {

	for i := 1; i <= 100; i++ {
		wg.Add(1)
		sendRequest()
		since++
	}

	wg.Wait()
}
