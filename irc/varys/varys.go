// Package varys is an abstraction that allows you to add or remove puppets,
// and receive a snapshot of state via an RPC-based interface.
//
// Why "varys"? Because it is the Master of Whisperers.
package varys

import "fmt"

type Varys struct {
	connConfig SetupParams
}

type Client interface {
	Setup(params SetupParams) error
	// Connect(uid string, params ConnectParams) (err error)
}

type SetupParams struct {
	UseTLS             bool // Whether we should use TLS
	InsecureSkipVerify bool // Controls tls.Config.InsecureSkipVerify, if using TLS

	ServerPassword string
	WebIRCPassword string
}

func (v *Varys) Setup(params SetupParams, _ *struct{}) error {
	fmt.Printf("setup params are now %#v", params)
	v.connConfig = params
	return nil
}

type ConnectParams struct {
	Nick     string
	Username string
	RealName string
	QuitMsg  string
}

// func (v *Varys) Connect(uid string, p)
