package server

import (
	"fmt"
	"log"
	"net"

	server "github.com/cloudwego/kitex/server"
	"github.com/lizuowang/gim/pkg/etcd_registry"
	rpc "github.com/lizuowang/gim/service/api/server/kitex_gen/im/rpc/imrpc"
	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	etcdRegistry *etcd_registry.ServiceRegister
	svr          server.Server
)

func InitServer(etc *clientv3.Client, serverName, ipAddr string) {

	var err error

	addr, _ := net.ResolveTCPAddr("tcp", ipAddr)
	svr = rpc.NewServer(new(ImRpcImpl), server.WithServiceAddr(addr))

	go startServer()

	name := fmt.Sprintf("%s%s", serverName, ipAddr)
	etcdRegistry, err = etcd_registry.NewServiceRegister(etc, name, ipAddr, 5)
	if err != nil {
		panic(fmt.Errorf("初始化etcd注册失败: %s ", err))
	}

	go etcdRegistry.ListenLeaseRespChan(func(leaseKeepResp *clientv3.LeaseKeepAliveResponse) {
	})

	log.Printf("InitServer 启动rpc server 服务注册 %s \n", ipAddr)
}

func startServer() {
	err := svr.Run()

	if err != nil {
		log.Println("startServer 启动rpc server 失败", err)
		panic(err)
	}
}

func OnShutdown() {
	log.Println("rpcServer 关闭")
	etcdRegistry.Close()
	svr.Stop()
}
