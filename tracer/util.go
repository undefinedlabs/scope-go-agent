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
	rndBytes := make([]byte, 8)
	seededIDGen.Read(rndBytes)
	uuidBytes := append(make([]byte, 8), rndBytes...)
	tid, _ := uuid.FromBytes(uuidBytes)
	return tid, uint64(seededIDGen.Int63())
}
