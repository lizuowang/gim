// Code generated by Kitex v0.11.3. DO NOT EDIT.
package imrpc

import (
	server "github.com/cloudwego/kitex/server"
	rpc "github.com/lizuowang/gim/service/api/server/kitex_gen/im/rpc"
)

// NewServer creates a server.Server with the given handler and options.
func NewServer(handler rpc.ImRpc, opts ...server.Option) server.Server {
	var options []server.Option

	options = append(options, opts...)
	options = append(options, server.WithCompatibleMiddlewareForUnary())

	svr := server.NewServer(options...)
	if err := svr.RegisterService(serviceInfo(), handler); err != nil {
		panic(err)
	}
	return svr
}

func RegisterService(svr server.Server, handler rpc.ImRpc, opts ...server.RegisterOption) error {
	return svr.RegisterService(serviceInfo(), handler, opts...)
}