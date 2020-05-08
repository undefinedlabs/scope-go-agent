package util

import (
	"bytes"
	"golang.org/x/text/unicode/norm"
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
		return unicode.IsControl(r)
	})
	return norm.NFKC.String(str)
}
