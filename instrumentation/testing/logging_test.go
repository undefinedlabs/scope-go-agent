package testing

import (
	"fmt"
	stdlog "log"
	"regexp"
	goTest "testing"
)

func TestLoggingRegex(t *goTest.T) {
	re := regexp.MustCompile(fmt.Sprintf(LOG_REGEX_TEMPLATE, stdlog.Prefix()))

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
	}

	for i := 0; i < len(loglines); i++ {
		logline := loglines[i]
		t.Run(fmt.Sprintf("Value %v", i), func(st *goTest.T) {
			matches := re.FindStringSubmatch(logline[0])

			for idx := 0; idx <= 5; idx++ {
				if matches[idx] != logline[idx] {
					t.Fatalf("Error in line[%v]: %v. Value expected: %v, Actual value: %v", i, logline[0], logline[idx], matches[idx])
				}
			}
		})
	}
}
