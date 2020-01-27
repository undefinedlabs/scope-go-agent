// +build !go1.14

package testing

import (
	"io"
	"sync"
	"time"
)

// clone of testing.common struct
type goCommon struct {
	mu         sync.RWMutex
	output     []byte
	w          io.Writer
	ran        bool
	failed     bool
	skipped    bool
	done       bool
	helpers    map[string]struct{}
	chatty     bool
	finished   bool
	hasSub     int32
	raceErrors int
	runner     string
	parent     *goCommon
	level      int
	creator    []uintptr
	name       string
	start      time.Time
	duration   time.Duration
	barrier    chan bool
	signal     chan bool
	sub        []*goT
}
