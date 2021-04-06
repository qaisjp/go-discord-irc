package varys

import (
	"net"

	"github.com/cenkalti/rpc2"
)

func NewServer() {
	varys := NewVarys()

	srv := rpc2.NewServer()
	srv.Handle("Varys.Setup", varys.Setup)
	srv.Handle("Varys.GetUIDToNicks", varys.GetUIDToNicks)
	srv.Handle("Varys.Connect", varys.Connect)
	srv.Handle("Varys.QuitIfConnected", varys.QuitIfConnected)
	srv.Handle("Varys.SendRaw", varys.SendRaw)
	srv.Handle("Varys.Nick", varys.Nick)
	srv.Handle("Varys.GetNick", varys.GetNick)
	srv.Handle("Varys.Connected", varys.Connected)

	lis, _ := net.Listen("tcp", "localhost:1234")
	srv.Accept(lis)
}
