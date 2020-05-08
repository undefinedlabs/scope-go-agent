package util

import (
	"bytes"
	"strings"
	"unicode"
)

func SanitizeString(value string) string {
	return SanitizeByteString([]byte(value))
}

func SanitizeByteString(value []byte) string {
	str := string(bytes.Runes(value))
	str = strings.TrimFunc(str, func(r rune) bool {
		if r == '\t' {
			return false
		}
		return unicode.IsControl(r) || r == '\u0000'
	})
	return str
}
