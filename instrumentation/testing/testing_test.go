package testing

import (
	"github.com/opentracing/opentracing-go"
	"github.com/opentracing/opentracing-go/log"
	"testing"
)

func TestExtractTestLogBuffer(t *testing.T) {
	test := StartTest(t)
	defer test.End()

	logLines := []string{
		"Hello World",
		"Hello        World     With         Spaces",
		"Hello\n World\nMulti\n        Line",
	}

	tester := &spanTester{}
	test.span = tester

	for _, item := range logLines {
		t.Log(item)
	}

	test.extractTestLoggerOutput()

	for i := 0; i < len(logLines); i++ {
		logItem := tester.logFields[i]

		if logItem[3].Value() != logLines[i] {
			t.FailNow()
		}
	}
}

type spanTester struct {
	logFields [][]log.Field
}

func (s *spanTester) Finish()                                          {}
func (s *spanTester) FinishWithOptions(opts opentracing.FinishOptions) {}
func (s *spanTester) Context() opentracing.SpanContext {
	return nil
}
func (s *spanTester) SetOperationName(operationName string) opentracing.Span {
	return nil
}
func (s *spanTester) SetTag(key string, value interface{}) opentracing.Span {
	return nil
}
func (s *spanTester) LogFields(fields ...log.Field) {
	s.logFields = append(s.logFields, fields)
}
func (s *spanTester) LogKV(alternatingKeyValues ...interface{}) {}
func (s *spanTester) SetBaggageItem(restrictedKey, value string) opentracing.Span {
	return nil
}
func (s *spanTester) BaggageItem(restrictedKey string) string {
	return ""
}
func (s *spanTester) Tracer() opentracing.Tracer {
	return nil
}
func (s *spanTester) LogEvent(event string)                                 {}
func (s *spanTester) LogEventWithPayload(event string, payload interface{}) {}
func (s *spanTester) Log(data opentracing.LogData)                          {}
