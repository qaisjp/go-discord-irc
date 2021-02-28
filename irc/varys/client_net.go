package varys

import (
	"log"
	"net/rpc"
)

type netClient struct {
	client *rpc.Client
}

func NewNetClient() Client {
	client, err := rpc.DialHTTP("tcp", "localhost:1234")
	if err != nil {
		log.Fatal("dialing:", err)
	}

	return &netClient{client: client}
}

func (c *netClient) Setup(params SetupParams) error {
	var reply struct{}
	return c.client.Call("Varys.Setup", params, &reply)
}

func (c *netClient) GetUIDToNicks() (result map[string]string, err error) {
	err = c.client.Call("Varys.GetUIDToNicks", struct{}{}, &result)
	return
}

func (c *netClient) Connect(params ConnectParams) error {
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

func (c *netClient) GetNick(uid string, nick *GetNickParams) error {
	return c.client.Call("Varys.GetNick", uid, nick)
}

func (c *netClient) Connected(uid string, connected *ConnectedParams) error {
	return c.client.Call("Varys.GetNick", uid, connected)
}
