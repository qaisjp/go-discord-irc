package bridge

import (
	"crypto/tls"

	"github.com/thoj/go-ircevent"
)

// TOOD: Insecure TLS!
func setupIRCConnection(con *irc.Connection) {
	con.UseTLS = true
	con.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: REALLY, THIS IS NOT A VERIFIED CONNECTION!
}
