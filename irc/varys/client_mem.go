package varys

import (
	"net"

	"github.com/akutz/memconn"
	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"
)

type memClient struct {
	varys *Varys
}

// NewMemClient returns an in-memory variant of varys
func NewMemClient(callback func(uid string, e *irc.Event)) Client {
	conn, err := memconn.Dial("memb", "varys")
	if err != nil {
		log.Fatal("dialing mem:", err)
	}

	return NewClient(conn, callback)
}

func NewNetClient(host string, callback func(uid string, e *irc.Event)) Client {
	conn, err := net.Dial("tcp", host)
	if err != nil {
		log.Fatal("dialing net:", err)
	}
	return NewClient(conn, callback)
}
