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

func (c *netClient) AddPuppet(name string) (realname string, err error) {
	err = c.client.Call("Varys.AddPuppet", name, &realname)
	return
}
