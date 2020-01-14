package testing

import (
	"strings"
	"testing"
)

func TestLogBufferRegex(t *testing.T) {
	test := StartTest(t)
	defer test.End()

	logLines := []string{
		"Hello World",
		"Hello        World     With         Spaces",
		"Hello\n World\nMulti\n        Line",
	}

	for _, item := range logLines {
		t.Log(item)
	}

	logBuffer := extractTestOutput(t)
	logs := string(*logBuffer)
	for idx, matches := range TESTING_LOG_REGEX.FindAllStringSubmatch(logs, -1) {
		message := strings.ReplaceAll(matches[3], "\n        ", "\n")
		if logLines[idx] != message {
			t.FailNow()
		}
	}
}

func TestExtractSubTestLogBuffer(t *testing.T) {
	t.Run("SubTest", TestLogBufferRegex)
}
