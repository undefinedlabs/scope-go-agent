package config

import (
	"fmt"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"sync"
)

type (
	TestDescription struct {
		Suite string
		Name  string
	}
)

var (
	testsToSkip map[string]TestDescription

	m sync.Mutex
)

// Gets the map of cached tests
func GetCachedTestsMap() map[string]TestDescription {
	m.Lock()
	defer m.Unlock()

	if testsToSkip != nil {
		return testsToSkip
	}

	config := instrumentation.GetRemoteConfiguration()
	testsToSkip = map[string]TestDescription{}
	if config != nil {
		if iCached, ok := config["cached"]; ok {
			cachedTests := iCached.([]interface{})
			for _, item := range cachedTests {
				testItem := item.(map[string]interface{})
				suite := fmt.Sprint(testItem["test_suite"])
				name := fmt.Sprint(testItem["test_name"])
				testFqn := fmt.Sprintf("%s.%s", suite, name)
				testsToSkip[testFqn] = TestDescription{
					Suite: suite,
					Name:  name,
				}
			}
		}
	}
	return testsToSkip
}
