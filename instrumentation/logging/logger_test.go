package logging

import (
	"fmt"
	stdlog "log"
	"regexp"
	"testing"
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
