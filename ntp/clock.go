package ntp

import (
	"fmt"
	"github.com/beevik/ntp"
	"sync"
	"time"
)

const (
	Server = "pool.ntp.org"
)

var (
	ntpOffset     time.Duration
	ntpOffsetOnce sync.Once
)

func init() {
	ntpOffsetOnce.Do(func() {
		response, err := ntp.Query(Server)
		if err != nil {
			fmt.Printf("%v\n", err)
		}
		ntpOffset = response.ClockOffset
	})
}

// Returns the time.Now() with the ntp offset
func Now() time.Time {
	return time.Now().Add(ntpOffset)
}
