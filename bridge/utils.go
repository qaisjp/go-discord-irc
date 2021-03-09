package bridge

import (
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/pkg/errors"
)

// Leftpad is from github.com/douglarek/leftpad
func Leftpad(s string, length int, ch ...rune) string {
	c := ' '
	if len(ch) > 0 {
		c = ch[0]
	}
	l := length - utf8.RuneCountInString(s)
	if l > 0 {
		s = strings.Repeat(string(c), l) + s
	}
	return s
}

// SnowflakeToIP takes a snowflake and the first half of an IP to make an IP suitable for WEBIRC
func SnowflakeToIP(base string, snowflake string) string {
	num, err := strconv.ParseUint(snowflake, 10, 64)
	if err != nil {
		panic(errors.Wrap(err, "could not convert snowflake to uint"))
	}

	for i, c := range Leftpad(strconv.FormatUint(num, 16), 16, '0') {
		if (i % 4) == 0 {
			base += ":"
		}
		base += string(c)
	}

	return base
}

// TruncateString is derived from https://github.com/gohugoio/hugo/blob/a03c631c420a03f9d90699abdf9be7e4fca0ff61/tpl/strings/truncate.go#L43
func TruncateString(length int, text string) string {
	ellipsis := " …"

	if utf8.RuneCountInString(text) <= length {
		return text
	}

	var lastWordIndex, lastNonSpace, currentLen, endTextPos int

	for i, r := range text {
		currentLen++
		if unicode.IsSpace(r) {
			lastWordIndex = lastNonSpace
		} else if unicode.In(r, unicode.Han, unicode.Hangul, unicode.Hiragana, unicode.Katakana) {
			lastWordIndex = i
		} else {
			lastNonSpace = i + utf8.RuneLen(r)
		}

		if currentLen > length {
			if lastWordIndex == 0 {
				endTextPos = i
			} else {
				endTextPos = lastWordIndex
			}
			out := text[0:endTextPos]

			return out + ellipsis
		}
	}

	return text
}
