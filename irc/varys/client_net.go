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
