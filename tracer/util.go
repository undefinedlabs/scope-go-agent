package tracer

import (
	cryptorand "crypto/rand"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"
)

var (
	seededIDGen = rand.New(rand.NewSource(generateSeed()))
	// The golang rand generators are *not* intrinsically thread-safe.
	seededIDLock sync.Mutex
)

func generateSeed() int64 {
	var b [8]byte
	_, err := cryptorand.Read(b[:])
	if err != nil {
		// Cannot seed math/rand package with cryptographically secure random number generator
		// Fallback to time.Now()
		return time.Now().UnixNano()
	}

	return int64(binary.LittleEndian.Uint64(b[:]))
}

func randomID() uint64 {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63())
}

func randomID2() (uint64, uint64) {
	seededIDLock.Lock()
	defer seededIDLock.Unlock()
	return uint64(seededIDGen.Int63()), uint64(seededIDGen.Int63())
}
