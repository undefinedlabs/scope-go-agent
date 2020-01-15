package testing

import (
	"testing"
)

func TestLogBufferRegex(t *testing.T) {
	test := StartTest(t)
	defer test.End()

	expectedLogLines := []string{
		"Hello World",
		"Hello        World     With         Spaces",
		"Hello\n World\nMulti\n        Line",
	}

	for _, item := range expectedLogLines {
		t.Log(item)
	}

	logBuffer := extractTestOutput(t)
	logs := string(*logBuffer)
	for idx, matches := range findMatchesLogRegex(logs) {
		if expectedLogLines[idx] != matches[3] {
			t.FailNow()
		}
	}
}

func TestExtractSubTestLogBuffer(t *testing.T) {
	t.Run("SubTest", TestLogBufferRegex)
}
