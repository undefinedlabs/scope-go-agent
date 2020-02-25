package autoinstrument

import (
	"reflect"
	"sync"
	"testing"

	"github.com/undefinedlabs/go-mpatch"

	"go.undefinedlabs.com/scopeagent"
	"go.undefinedlabs.com/scopeagent/env"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
)

var (
	once sync.Once
)

func init() {
	once.Do(func() {
		var m *testing.M
		var mRunMethod reflect.Method
		var ok bool
		mType := reflect.TypeOf(m)
		if mRunMethod, ok = mType.MethodByName("Run"); !ok {
			return
		}

		var runPatch *mpatch.Patch
		var err error
		runPatch, err = mpatch.PatchMethodByReflect(mRunMethod, func(m *testing.M) int {
			logOnError(runPatch.Unpatch())
			defer func() {
				logOnError(runPatch.Patch())
			}()
			scopetesting.PatchTestingLogger()
			defer scopetesting.UnpatchTestingLogger()
			return scopeagent.Run(m)
		})
		logOnError(err)

		if !env.ScopeTestingDisableParallel.Value {
			return
		}
		var t *testing.T
		var tParallelMethod reflect.Method
		tType := reflect.TypeOf(t)
		if tParallelMethod, ok = tType.MethodByName("Parallel"); !ok {
			return
		}
		_, err = mpatch.PatchMethodByReflect(tParallelMethod, func(t *testing.T) {})
		logOnError(err)
	})
}

func logOnError(err error) {
	if err != nil {
		instrumentation.Logger().Println(err)
	}
}
