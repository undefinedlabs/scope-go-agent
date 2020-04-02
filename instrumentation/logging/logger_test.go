package logging

import (
	"fmt"
	stdlog "log"
	"os"
	"regexp"
	"testing"
	"time"

	"github.com/opentracing/opentracing-go"

	"go.undefinedlabs.com/scopeagent/tags"
)

func TestLoggingRegex(t *testing.T) {
	re := regexp.MustCompile(fmt.Sprintf(logRegexTemplate, stdlog.Prefix()))

	var loglines = [][]string{
		{"2009/01/23 01:23:23.123123 /a/b/c/d.go:23: message", "2009/01/23", "01:23:23.123123", "/a/b/c/d.go", "23", "message"},
		{"2009/01/23 01:23:23 /a/b/c/d.go:23: message", "2009/01/23", "01:23:23", "/a/b/c/d.go", "23", "message"},
		{"2009/01/23 /a/b/c/d.go:23: message", "2009/01/23", "", "/a/b/c/d.go", "23", "message"},
		{"/a/b/c/d.go:23: message", "", "", "/a/b/c/d.go", "23", "message"},

		{"2009/01/23 01:23:23.123123 d.go:23: message", "2009/01/23", "01:23:23.123123", "d.go", "23", "message"},
		{"2009/01/23 01:23:23 d.go:23: message", "2009/01/23", "01:23:23", "d.go", "23", "message"},
		{"2009/01/23 d.go:23: message", "2009/01/23", "", "d.go", "23", "message"},
		{"d.go:23: message", "", "", "d.go", "23", "message"},

		{"2009/01/23 01:23:23.123123 message", "2009/01/23", "01:23:23.123123", "", "", "message"},
		{"2009/01/23 01:23:23 message", "2009/01/23", "01:23:23", "", "", "message"},
		{"2009/01/23 message", "2009/01/23", "", "", "", "message"},
		{"message", "", "", "", "", "message"},

		{"2009/01/23 01:23:23.123123 c:/a/b/c/d.go:23: message", "2009/01/23", "01:23:23.123123", "c:/a/b/c/d.go", "23", "message"},
		{"2009/01/23 01:23:23 c:/a/b/c/d.go:23: message", "2009/01/23", "01:23:23", "c:/a/b/c/d.go", "23", "message"},
		{"2009/01/23 c:/a/b/c/d.go:23: message", "2009/01/23", "", "c:/a/b/c/d.go", "23", "message"},
		{"c:/a/b/c/d.go:23: message", "", "", "c:/a/b/c/d.go", "23", "message"},

		{"2009/01/23 01:23:23.123123 c:\\a\\b\\c\\d.go:23: message", "2009/01/23", "01:23:23.123123", "c:\\a\\b\\c\\d.go", "23", "message"},
		{"2009/01/23 01:23:23 c:\\a\\b\\c\\d.go:23: message", "2009/01/23", "01:23:23", "c:\\a\\b\\c\\d.go", "23", "message"},
		{"2009/01/23 c:\\a\\b\\c\\d.go:23: message", "2009/01/23", "", "c:\\a\\b\\c\\d.go", "23", "message"},
		{"c:\\a\\b\\c\\d.go:23: message", "", "", "c:\\a\\b\\c\\d.go", "23", "message"},

		{"2009/01/23 01:23:23.123123 Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.",
			"2009/01/23", "01:23:23.123123", "", "",
			"Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."},

		{"2009/01/23 01:23:23.123123 Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum.",
			"2009/01/23", "01:23:23.123123", "", "",
			"Lorem Ipsum is simply dummy text of the printing and typesetting industry. Lorem Ipsum has been the industry's standard dummy text ever since the 1500s, when an unknown printer took a galley of type and scrambled it to make a type specimen book. It has survived not only five centuries, but also the leap into electronic typesetting, remaining essentially unchanged. It was popularised in the 1960s with the release of Letraset sheets containing Lorem Ipsum passages, and more recently with desktop publishing software like Aldus PageMaker including versions of Lorem Ipsum."},
	}

	for i := 0; i < len(loglines); i++ {
		logline := loglines[i]
		t.Run(fmt.Sprintf("Value %v", i), func(st *testing.T) {
			matches := re.FindStringSubmatch(logline[0])

			for idx := 0; idx <= 5; idx++ {
				if matches[idx] != logline[idx] {
					t.Fatalf("Error in line[%v]: %v. Value expected: %v, Actual value: %v", i, logline[0], logline[idx], matches[idx])
				}
			}
		})
	}
}

func TestStandardLoggerInstrumentation(t *testing.T) {
	stdlog.SetFlags(stdlog.LstdFlags | stdlog.Lmicroseconds | stdlog.Llongfile)
	PatchStandardLogger()
	defer UnpatchStandardLogger()

	messages := []string{"Print log", "Println log"}

	stdlog.Print(messages[0])
	stdlog.Println(messages[1])

	checkRecords(t, messages)
}

func TestCustomLoggerInstrumentation(t *testing.T) {
	logger := stdlog.New(os.Stdout, "", stdlog.LstdFlags|stdlog.Lmicroseconds|stdlog.Llongfile)
	PatchLogger(logger)
	defer UnpatchLogger(logger)

	messages := []string{"Print log", "Println log"}

	logger.Print(messages[0])
	logger.Println(messages[1])

	checkRecords(t, messages)
}

func TestStdOutInstrumentation(t *testing.T) {
	PatchStdOut()
	defer UnpatchStdOut()
	Reset()

	messages := []string{"Println log", "Println log"}

	fmt.Println(messages[0])
	fmt.Println(messages[1])

	<-time.After(time.Second)

	checkRecords(t, messages)
}

func checkRecords(t *testing.T, messages []string) {
	records := GetRecords()

	if len(records) != 2 {
		t.Fatalf("error in the number of records, expected: 2, actual: %d", len(records))
	}

	for _, msg := range messages {
		if !checkMessage(msg, records) {
			t.Fatalf("the message '%s' was not in the log records", msg)
		}
	}
}

func checkMessage(msg string, records []opentracing.LogRecord) bool {
	for _, rec := range records {
		for _, field := range rec.Fields {
			if field.Key() == tags.EventMessage {
				if msg == fmt.Sprint(field.Value()) {
					return true
				}
			}
		}
	}
	return false
}

func BenchmarkPatchStandardLogger(b *testing.B) {
	for i:=0; i < b.N; i++ {
		PatchStandardLogger()
		UnpatchStandardLogger()
	}
}

var lg *stdlog.Logger
func BenchmarkPatchLogger(b *testing.B) {
	for i:=0; i < b.N; i++ {
		lg = stdlog.New(os.Stdout, "", stdlog.Llongfile | stdlog.Lmicroseconds)
		PatchLogger(lg)
		UnpatchLogger(lg)
	}
}