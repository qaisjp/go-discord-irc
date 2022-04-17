package varys

import (
	"net"

	irc "github.com/qaisjp/go-ircevent"
	log "github.com/sirupsen/logrus"

	"github.com/cenkalti/rpc2"
)

type netClient struct {
	client   *rpc2.Client
	callback func(string, *irc.Event)
}

func NewClient(conn net.Conn, callback func(uid string, e *irc.Event)) Client {
	client := &netClient{
		client:   rpc2.NewClient(conn),
		callback: callback,
	}

	client.client.Handle("Varys.Callback", func(_ *rpc2.Client, e *Event, _ *struct{}) error {
		log.Printf("Callback received on the client: %#v\n", e)
		callback(e.VarysUID, e.toReal())
		return nil
	})

	go client.client.Run()

	return client
}

func (c *netClient) Setup(params SetupParams) error {
	var reply struct{}
	// fmt.Println("setup called")
	err := c.client.Call("Varys.Setup", params, &reply)
	// fmt.Println("setup returned", err)
	return err
}

func (c *netClient) GetUIDToNicks() (result map[string]string, err error) {
	err = c.client.Call("Varys.GetUIDToNicks", struct{}{}, &result)
	return
}

func (c *netClient) Connect(params ConnectParams) error {
	// fmt.Println("connect called")

	var reply struct{}
	return c.client.Call("Varys.Connect", params, &reply)
}

func (c *netClient) QuitIfConnected(uid string, quitMessage string) error {
	var reply struct{}
	return c.client.Call("Varys.QuitIfConnected", QuitParams{uid, quitMessage}, &reply)
}

func (c *netClient) SendRaw(uid string, params InterpolationParams, messages ...string) error {
	var reply struct{}
	return c.client.Call("Varys.SendRaw", SendRawParams{uid, messages, params}, &reply)
}

func (c *netClient) Nick(uid string, nick string) error {
	var reply struct{}
	return c.client.Call("Varys.Nick", NickParams{uid, nick}, &reply)
}

func (c *netClient) GetNick(uid string) (result string, err error) {
	err = c.client.Call("Varys.GetNick", uid, &result)
	return
}

func (c *netClient) Connected(uid string) (result bool, err error) {
	err = c.client.Call("Varys.Connected", uid, &result)
	return
}
