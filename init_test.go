package scopeagent

import (
	"fmt"
	"os"
	"testing"

	"go.undefinedlabs.com/scopeagent/agent"
)

func TestMain(m *testing.M) {
	os.Exit(Run(m,
		agent.WithRetriesOnFail(3),
		agent.WithHandlePanicAsFail()))
}

func TestMultiple(t *testing.T) {
	test := GetTest(t)
	for i := 0; i < 100000; i++ {
		test.Run(fmt.Sprintf("%v", i), func(t *testing.T) {
			t.Parallel()
		})
	}
}
