// Package varys is an abstraction that allows you to add or remove puppets,
// and receive a snapshot of state via an RPC-based interface.
//
// Why "varys"? Because it is the Master of Whisperers.
package varys

import (
	"crypto/tls"
	"fmt"
	"log"
	"strings"

	"github.com/cenkalti/rpc2"
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
	// GetNick gets the current connection's nick
	GetNick(uid string) (string, error)
	// Connected returns the status of the current connection
	Connected(uid string) (bool, error)
}

type SetupParams struct {
	UseTLS             bool // Whether we should use TLS
	InsecureSkipVerify bool // Controls tls.Config.InsecureSkipVerify, if using TLS

	Server         string
	ServerPassword string
	WebIRCPassword string
}

func (v *Varys) Setup(client *rpc2.Client, params SetupParams, _ *struct{}) error {
	v.connConfig = params
	return nil
}

func (v *Varys) GetUIDToNicks(client *rpc2.Client, _ struct{}, result *map[string]string) error {
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

	// Event codes to subscribe and send to the master callback
	Callbacks []string
}

func (v *Varys) Connect(client *rpc2.Client, params ConnectParams, _ *struct{}) error {
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

	for _, eventcode := range params.Callbacks {
		conn.AddCallback(eventcode, func(e *irc.Event) {
			varysEvent := eventFomReal(params.UID, e)

			var reply struct{}
			if err := client.Call("Varys.Callback", varysEvent, &reply); err != nil {
				log.Fatalln("Failed to call Varys.Callback:", err.Error())
			}
		})
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

func (v *Varys) QuitIfConnected(client *rpc2.Client, params QuitParams, _ *struct{}) error {
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

func (v *Varys) SendRaw(client *rpc2.Client, params SendRawParams, _ *struct{}) error {
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

func (v *Varys) GetNick(client *rpc2.Client, uid string, result *string) error {
	if conn, ok := v.uidToConns[uid]; ok {
		*result = conn.GetNick()
	}
	return nil
}

func (v *Varys) Connected(client *rpc2.Client, uid string, result *bool) error {
	if conn, ok := v.uidToConns[uid]; ok {
		*result = conn.Connected()
	}

	return nil
}

type NickParams struct {
	UID  string
	Nick string
}

func (v *Varys) Nick(client *rpc2.Client, params NickParams, _ *struct{}) error {
	if conn, ok := v.uidToConns[params.UID]; ok {
		conn.Nick(params.Nick)
	}
	return nil
}
