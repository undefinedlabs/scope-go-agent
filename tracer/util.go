package tracer

import (
	cryptorand "crypto/rand"
	"encoding/hex"
	"math"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"go.undefinedlabs.com/scopeagent/instrumentation"
)

var (
	random *rand.Rand
	mu     sync.Mutex
)

func getRandomId() uint64 {
	mu.Lock()
	defer mu.Unlock()
	ensureRandom()
	return random.Uint64()
}

func getRandomUUID() uuid.UUID {
	mu.Lock()
	defer mu.Unlock()
	ensureRandom()
	return uuid.New()
}

func ensureRandom() {
	if random == nil {
		random = rand.New(&safeSource{
			source: rand.NewSource(getSeed()),
		})
		uuid.SetRand(random)
	}
}

//go:noinline
func getSeed() int64 {
	var seed int64
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(math.MaxInt64))
	if err == nil {
		seed = n.Int64()
	} else {
		instrumentation.Logger().Printf("cryptorand error generating seed: %v. \n falling back to time.Now()", err)

		// Adding some jitter to the clock seed using golang channels and goroutines
		jitterStart := time.Now()
		cb := make(chan time.Time, 0)
		go func() { cb <- <-time.After(time.Nanosecond) }()
		now := <-cb
		jitter := time.Since(jitterStart)

		// Seed based on the clock + some jitter
		seed = now.Add(jitter).UnixNano()
	}
	instrumentation.Logger().Printf("seed: %d", seed)
	return seed
}

// safeSource holds a thread-safe implementation of rand.Source64.
type safeSource struct {
	source rand.Source
	sync.Mutex
}

func (rs *safeSource) Int63() int64 {
	rs.Lock()
	n := rs.source.Int63()
	rs.Unlock()

	return n
}

func (rs *safeSource) Uint64() uint64 { return uint64(rs.Int63()) }

func (rs *safeSource) Seed(seed int64) {
	rs.Lock()
	rs.source.Seed(seed)
	rs.Unlock()
}

func UUIDToString(uuid uuid.UUID) string {
	if val, err := uuid.MarshalBinary(); err != nil {
		panic(err)
	} else {
		return hex.EncodeToString(val)
	}
}

func StringToUUID(val string) (uuid.UUID, error) {
	if data, err := hex.DecodeString(val); err != nil {
		return uuid.UUID{}, err
	} else {
		res := uuid.UUID{}
		err := res.UnmarshalBinary(data)
		return res, err
	}
}
