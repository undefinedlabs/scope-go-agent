package config

import (
	"fmt"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"sync"
)

var (
	testsToSkip map[string]struct{}

	m sync.Mutex
)

func GetCachedTestsMap() map[string]struct{} {
	m.Lock()
	defer m.Unlock()

	if testsToSkip != nil {
		return testsToSkip
	}

	config := instrumentation.GetRemoteConfiguration()
	testsToSkip = map[string]struct{}{}
	if iCached, ok := config["cached"]; ok {
		cachedTests := iCached.([]interface{})
		for _, item := range cachedTests {
			testItem := item.(map[string]interface{})
			testFqn := fmt.Sprintf("%v.%v", testItem["test_suite"], testItem["test_name"])
			testsToSkip[testFqn] = struct{}{}
		}
	}
	return testsToSkip
}
