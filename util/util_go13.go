// +build go1.13

package util

import (
	"bytes"
	"strings"
)

func BytesToValidUTF8(s, replacement []byte) []byte {
	return bytes.ToValidUTF8(s, replacement)
}

func StringToValidUTF8(s, replacement string) string {
	return strings.ToValidUTF8(s, replacement)
}
