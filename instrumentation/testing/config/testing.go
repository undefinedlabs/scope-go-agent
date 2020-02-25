package config

import (
	"sync"
)

var (
	testsToSkip map[string]struct{}

	m sync.RWMutex
)

func SetFqnToSkip(fqns ...string) {
	m.Lock()
	defer m.Unlock()

	testsToSkip = map[string]struct{}{}
	for _, val := range fqns {
		testsToSkip[val] = struct{}{}
	}
}

func GetSkipMap() map[string]struct{} {
	m.RLock()
	defer m.RUnlock()

	return testsToSkip
}
