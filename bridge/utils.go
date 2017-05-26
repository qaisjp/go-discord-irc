package bridge

import (
	"crypto/tls"

	"github.com/thoj/go-ircevent"
)

func setupIRCConnection(con *irc.Connection) {
	// con.VerboseCallbackHandler = true
	con.Debug = true
	con.UseTLS = true
	con.TLSConfig = &tls.Config{InsecureSkipVerify: true} // TODO: Insecure TLS!
}
