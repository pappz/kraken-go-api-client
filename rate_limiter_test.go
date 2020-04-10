package krakenapi

import (
	"log"
	"sync"
	"testing"
)

var (
	wg sync.WaitGroup
)
func sendRequest() {

	_, err := publicAPI.Trades("XXBTZCAD", 1546470333801400000)
	if err != nil {
		log.Fatalf("%v", err)
	}
	wg.Done()
}

func TestNewPublicRateLimit(t *testing.T) {

	for i := 1; i <= 50; i++ {
		wg.Add(1)
		go sendRequest()
	}

	wg.Wait()
}
