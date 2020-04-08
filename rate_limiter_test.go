package krakenapi

import (
	"fmt"
	"net/url"
	"strconv"
	"testing"
)

func TestNewPublicRateLimit(t *testing.T) {
	rl := NewPublicRateLimit()
	myUrl := fmt.Sprintf("%s/%s/public/%s", APIURL, APIVersion, "Trades")
	values := url.Values{"pair": {"XXBTZEUR"}}
	values.Set("since", strconv.FormatInt(1495777604391411290, 10))

	_, err := rl.limitedRequest(publicAPI, myUrl, values, nil, nil)
	if err != nil {
		t.Fatalf("%v", err)
	}
}
