package util

import (
	"strings"
	"unicode"
)

func RemoveNonGraphicChars(value string) string {
	return strings.TrimFunc(value, func(r rune) bool {
		return !unicode.IsGraphic(r)
	})
}
