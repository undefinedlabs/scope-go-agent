package ntp

import (
	"fmt"
	"github.com/beevik/ntp"
	"time"
)

const (
	Server = "pool.ntp.org"
)

var (
	ntpOffset time.Duration
)

func init() {
	tries := 3
	var response *ntp.Response
	for {
		if tries > 0 {
			tries--
			r, err := ntp.Query(Server)
			if err != nil {
				fmt.Printf("%v\n", err)
				time.Sleep(1 * time.Second)
				continue
			}
			response = r
		}
		break
	}
	if response != nil {
		ntpOffset = response.ClockOffset
	} else {
		fmt.Println("Error getting the NTP offset")
	}
}

// Returns the time.Now() with the ntp offset
func Now() time.Time {
	return time.Now().Add(ntpOffset)
}
