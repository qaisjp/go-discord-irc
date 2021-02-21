// Package varys is an abstraction that allows you to add or remove puppets,
// and receive a snapshot of state via an RPC-based interface.
//
// Why "varys"? Because it is the Master of Whisperers.
package varys

import (
	"crypto/tls"
	"fmt"
	"strings"

	irc "github.com/qaisjp/go-ircevent"
)

type Varys struct {
	connConfig SetupParams
	uidToConns map[string]*irc.Connection
}

func NewVarys() *Varys {
	return &Varys{uidToConns: make(map[string]*irc.Connection)}
}

func (v *Varys) connCall(uid string, fn func(*irc.Connection)) {
	if uid == "" {
		for _, conn := range v.uidToConns {
			fn(conn)
		}
		return
	}

	if conn, ok := v.uidToConns[uid]; ok {
		fn(conn)
	}
}

type Client interface {
	Setup(params SetupParams) error
	GetUIDToNicks() (map[string]string, error)
	Connect(params ConnectParams) error // Does not yet support netClient
	QuitIfConnected(uid string, quitMsg string) error
	Nick(uid string, nick string) error

	// SendRaw supports a blank uid to send to all connections.
	SendRaw(uid string, params InterpolationParams, messages ...string) error
}

type SetupParams struct {
	UseTLS             bool // Whether we should use TLS
	InsecureSkipVerify bool // Controls tls.Config.InsecureSkipVerify, if using TLS

	Server         string
	ServerPassword string
	WebIRCPassword string
}

func (v *Varys) Setup(params SetupParams, _ *struct{}) error {
	v.connConfig = params
	return nil
}

func (v *Varys) GetUIDToNicks(_ struct{}, result *map[string]string) error {
	conns := v.uidToConns
	m := make(map[string]string, len(conns))
	for uid, conn := range conns {
		m[uid] = conn.GetNick()
	}
	*result = m
	return nil
}

type ConnectParams struct {
	UID string

	Nick     string
	Username string
	RealName string

	WebIRCSuffix string

	// TODO(qaisjp): does not support net/rpc!!!!
	Callbacks map[string]func(*irc.Event)
}

func (v *Varys) Connect(params ConnectParams, _ *struct{}) error {
	conn := irc.IRC(params.Nick, params.Username)
	// conn.Debug = true
	conn.RealName = params.RealName

	// TLS things, and the server password
	conn.Password = v.connConfig.ServerPassword
	conn.UseTLS = v.connConfig.UseTLS
	if v.connConfig.InsecureSkipVerify {
		conn.TLSConfig = &tls.Config{
			InsecureSkipVerify: true,
		}
	}

	// Set up WebIRC, if a suffix is provided
	if params.WebIRCSuffix != "" {
		conn.WebIRC = v.connConfig.WebIRCPassword + " " + params.WebIRCSuffix
	}

	// On kick, rejoin the channel
	conn.AddCallback("KICK", func(e *irc.Event) {
		if e.Arguments[1] == conn.GetNick() {
			conn.Join(e.Arguments[0])
		}
	})

	for eventcode, callback := range params.Callbacks {
		conn.AddCallback(eventcode, callback)
	}

	err := conn.Connect(v.connConfig.Server)
	if err != nil {
		return fmt.Errorf("error opening irc connection: %w", err)
	}

	v.uidToConns[params.UID] = conn
	go conn.Loop()
	return nil
}

type QuitParams struct {
	UID         string
	QuitMessage string
}

func (v *Varys) QuitIfConnected(params QuitParams, _ *struct{}) error {
	if conn, ok := v.uidToConns[params.UID]; ok {
		if conn.Connected() {
			conn.QuitMessage = params.QuitMessage
			conn.Quit()
		}
	}
	delete(v.uidToConns, params.UID)
	return nil
}

type InterpolationParams struct {
	Nick bool
}
type SendRawParams struct {
	UID      string
	Messages []string

	Interpolation InterpolationParams
}

func (v *Varys) SendRaw(params SendRawParams, _ *struct{}) error {
	v.connCall(params.UID, func(c *irc.Connection) {
		for _, msg := range params.Messages {
			if params.Interpolation.Nick {
				msg = strings.ReplaceAll(msg, "${NICK}", c.GetNick())
			}
			c.SendRaw(msg)
		}
	})
	return nil
}

type NickParams struct {
	UID  string
	Nick string
}

func (v *Varys) Nick(params NickParams, _ *struct{}) error {
	if conn, ok := v.uidToConns[params.UID]; ok {
		conn.Nick(params.Nick)
	}
	return nil
}
