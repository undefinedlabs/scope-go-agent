package scopeagent // import "go.undefinedlabs.com/scopeagent"

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"testing"

	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/instrumentation"
	"go.undefinedlabs.com/scopeagent/instrumentation/logging"
	scopetesting "go.undefinedlabs.com/scopeagent/instrumentation/testing"
)

var defaultAgent *agent.Agent

// Helper function to run a `testing.M` object and gracefully stopping the agent afterwards
func Run(m *testing.M, opts ...agent.Option) int {
	opts = append(opts, agent.WithTestingModeEnabled())
	newAgent, err := agent.NewAgent(opts...)
	if err != nil {
		res := m.Run()
		fmt.Printf("\n%v\n", err)
		return res
	}

	logging.PatchStandardLogger()

	scopetesting.Init(m)

	// Handle SIGINT and SIGTERM
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigs
		instrumentation.Logger().Println("Terminating agent, sending partial results...")
		newAgent.Stop()
		os.Exit(1)
	}()

	defaultAgent = newAgent
	return newAgent.Run(m)
}

// Instruments the given test, returning a `Test` object that can be used to extend the test trace
func StartTest(t *testing.T, opts ...scopetesting.Option) *scopetesting.Test {
	pc, _, _, _ := runtime.Caller(1)
	return scopetesting.StartTestFromCaller(t, pc, opts...)
}

// Gets the *Test from a *testing.T
func GetTest(t *testing.T) *scopetesting.Test {
	return scopetesting.GetTest(t)
}

// Gets the context from a test
func GetContextFromTest(t *testing.T) context.Context {
	test := GetTest(t)
	if test != nil {
		return test.Context()
	}
	return context.TODO()
}

// Gets the *Benchmark from a *testing.B
func GetBenchmark(b *testing.B) *scopetesting.Benchmark {
	return scopetesting.GetBenchmark(b)
}

// Instruments the given benchmark func
//
// Example of usage:
//
// func factorial(value int) int {
//		if value == 1 {
//			return 1
//		}
//		return value * factorial(value-1)
// }
//
// func BenchmarkFactorial(b *testing.B) {
// 		scopeagent.StartBenchmark(b, func(b *testing.B) {
// 			for i := 0; i < b.N; i++ {
//				_ = factorial(25)
//			}
//		}
// }
//
// func BenchmarkFactorialWithSubBenchmarks(b *testing.B) {
//		res := 0
//
//		b.Run("25", func(b *testing.B) {
//			scopeagent.StartBenchmark(b, func(b *testing.B) {
//				for i := 0; i < b.N; i++ {
//					res = factorial(25)
//				}
//			})
//		})
//
//		b.Run("50", func(b *testing.B) {
//			scopeagent.StartBenchmark(b, func(b *testing.B) {
//				for i := 0; i < b.N; i++ {
//					res = factorial(50)
//				}
//			})
//		})
//
//		b.Run("75", func(b *testing.B) {
//			scopeagent.StartBenchmark(b, func(b *testing.B) {
//				for i := 0; i < b.N; i++ {
//					res = factorial(75)
//				}
//			})
//		})
//
//		b.Run("100", func(b *testing.B) {
//			scopeagent.StartBenchmark(b, func(b *testing.B) {
//				for i := 0; i < b.N; i++ {
//					res = factorial(100)
//				}
//			})
//		})
//
//		_ = res
// }
func StartBenchmark(b *testing.B, benchFunc func(b *testing.B)) {
	pc, _, _, _ := runtime.Caller(1)
	scopetesting.StartBenchmark(b, pc, benchFunc)
}
