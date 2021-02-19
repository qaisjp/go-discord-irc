package varys

import (
	"log"
	"net"
	"net/http"
	"net/rpc"
)

func NewServer() {
	varys := new(Varys)
	rpc.Register(varys)
	rpc.HandleHTTP()
	l, e := net.Listen("tcp", ":1234")
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}
