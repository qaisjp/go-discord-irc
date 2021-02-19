// Package varys is an abstraction that allows you to add or remove puppets,
// and receive a snapshot of state via an RPC-based interface.
//
// Why "varys"? Because it is the Master of Whisperers.
package varys

type Client interface {
	AddPuppet(name string) (realname string, err error)
}

type Varys struct{}

func (v *Varys) AddPuppet(name string, realname *string) error {
	*realname = name
	return nil
}
