package logrus

import (
	"github.com/sirupsen/logrus"
	"os"
	"testing"

	"go.undefinedlabs.com/scopeagent"
	"go.undefinedlabs.com/scopeagent/agent"
	"go.undefinedlabs.com/scopeagent/tracer"
)

var r *tracer.InMemorySpanRecorder

func TestMain(m *testing.M) {
	// Test tracer
	r = tracer.NewInMemoryRecorder()
	os.Exit(scopeagent.Run(m, agent.WithRecorders(r)))
}

func TestLogrus(t *testing.T) {
	ctx := scopeagent.GetContextFromTest(t)
	r.Reset()

	logger := logrus.New()
	logger.SetLevel(logrus.TraceLevel)
	logger.SetReportCaller(true)
	logger.AddHook(&ScopeHook{})

	logger.WithContext(ctx).WithField("Data", "Value").Error("Error message")
	logger.WithContext(ctx).WithField("Data", "Value").Warn("Warning message")
	logger.WithContext(ctx).WithField("Data", "Value").Info("Info message")
	logger.WithContext(ctx).WithField("Data", "Value").Debug("Debug message")
	logger.WithContext(ctx).WithField("Data", "Value").Trace("Trace message")
}
