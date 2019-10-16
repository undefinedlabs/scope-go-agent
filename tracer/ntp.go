package tracer

import (
	"sync"
	"time"

	"github.com/beevik/ntp"
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

func getNTPOffset() time.Duration {
	for i := 1; i <= retries; i++ {
		r, err := ntp.QueryWithOptions(server, ntp.QueryOptions{Timeout: timeout})
		if err == nil {
			return r.ClockOffset
		}
		time.Sleep(backoff)
	}
	return 0
}

// Calculates and saves the time offset using NTP
func CalculateNTPOffset() time.Duration {
	once.Do(func() {
		ntpOffset = getNTPOffset()
	})
	return ntpOffset
}

// Returns an NTP-adjusted time.Now()
func Now() time.Time {
	CalculateNTPOffset()
	return time.Now().Add(ntpOffset)
}
