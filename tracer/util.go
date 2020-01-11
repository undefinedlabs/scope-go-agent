package tracer

import (
	"github.com/google/uuid"
	"math/rand"
	"sync"
	"time"
)

var (
	seededIDGen = rand.New(rand.NewSource(time.Now().UnixNano()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

func randomID() uint64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63())
}

func randomID2() (uuid.UUID, uint64) {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uuid.New(), uint64(seededIDGen.Int63())
}
