package agent

import (
	"github.com/beevik/ntp"
	"sync"
	"time"
)

const (
	server  = "pool.ntp.org"
	retries = 5
	timeout = 1 * time.Second
	backoff = 1 * time.Second
)

var (
	ntpOffset time.Duration
	once      sync.Once
)

// Gets the NTP offset from the ntp server pool
func getNTPOffset() (time.Duration, error) {
	var ntpError error = nil
	for i := 1; i <= retries; i++ {
		r, err := ntp.QueryWithOptions(server, ntp.QueryOptions{Timeout: timeout})
		if err == nil {
			return r.ClockOffset, nil
		}
		ntpError = err
		time.Sleep(backoff)
	}
	return 0, ntpError
}
