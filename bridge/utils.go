package bridge

import (
	"crypto/tls"
	"strconv"

	"strings"
	"unicode/utf8"

	"github.com/pkg/errors"

	"github.com/thoj/go-ircevent"
)

// TOOD: Insecure TLS!
func setupIRCConnection(con *irc.Connection) {
	con.UseTLS = true
	con.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: REALLY, THIS IS NOT A VERIFIED CONNECTION!
}

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

// Takes a snowflake and the first half of an IP to make an IP suitable for WEBIRC
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
