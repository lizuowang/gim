package client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/cloudwego/kitex/client"
	"github.com/lizuowang/gim/pkg/etcd_registry"
	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lizuowang/gim/service/api/server/kitex_gen/im/rpc"
	"github.com/lizuowang/gim/service/api/server/kitex_gen/im/rpc/imrpc"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Client struct {
	Cli  imrpc.Client
	Addr string
}

var (
	ServiceName string
	clients     map[string]*Client
	lock        sync.RWMutex
)

func InitClient(serviceName string, etcdClient *clientv3.Client) {
	ServiceName = serviceName
	clients = make(map[string]*Client)
	// 发现服务
	etcdDiscovery := etcd_registry.NewServiceDiscovery(etcdClient, serviceName, onRegister, onUnregister)

	etcdDiscovery.WatchService()

	log.Printf("InitClient 启动rpc client 服务发现 %s \n", serviceName)

}

// 注册服务
func onRegister(kv *mvccpb.KeyValue) {
	addr := string(kv.Value)
	key := string(kv.Key)

	lock.Lock()
	defer lock.Unlock()
	oldClient, ok := clients[key]
	if ok {
		if oldClient.Addr == addr {
			return
		}
	}

	client, err := imrpc.NewClient(ServiceName, client.WithHostPorts(addr))
	if err != nil {
		logger.L.Error("api.onRegister new client error", zap.Error(err))
		return
	}
	clients[key] = &Client{
		Cli:  client,
		Addr: addr,
	}
	logger.L.Info("api.onRegister register client success", zap.String("key", key), zap.String("addr", addr))
}

// 注销服务
func onUnregister(kv *mvccpb.KeyValue) {
	key := string(kv.Key)

	lock.Lock()
	defer lock.Unlock()
	delete(clients, key)
	logger.L.Info("api.onUnregister unregister client success", zap.String("key", key))
}

// 根据地址获取客户端
func GetClientByAddr(addr string) (*Client, error) {
	key := ServiceName + addr
	lock.RLock()
	defer lock.RUnlock()

	client, ok := clients[key]
	if !ok {
		logger.L.Error("api.GetClientByAddr client not found", zap.String("addr", addr))
		return nil, im_err.NewImError(im_err.ErrRpcClientNotFound)
	}

	return client, nil
}

// 发送消息到用户
func SendMsgToUser(server *types.Server, uid string, sendMsg []byte) (res bool, err error) {
	client, err := GetClientByAddr(server.String())
	if err != nil {
		return
	}
	sendReq := &rpc.SendMsgReq{
		Uid: uid,
		Msg: sendMsg,
	}

	rsp, err := client.Cli.SendMsgToUser(context.Background(), sendReq)
	if err != nil {
		logger.L.Error("rpcClient.SendMsgToUser 发送消息失败", zap.String("addr", server.String()), zap.Any("error", err))
		return
	}

	if rsp.GetRetCode() != im_err.OK {
		logger.L.Error("rpcClient.SendMsgToUser 发送消息响应码错误", zap.String("addr", server.String()), zap.Int32("retCode", rsp.GetRetCode()))
		return
	}

	res = true
	return
}

// 发送消息到组内用户
func SendMsgToUserByTgid(server *types.Server, uid string, tgid string, sendMsg []byte) (res bool, err error) {
	client, err := GetClientByAddr(server.String())
	if err != nil {
		return
	}
	sendReq := &rpc.SendMsgByTgidReq{
		Uid:  uid,
		Tgid: tgid,
		Msg:  sendMsg,
	}

	rsp, err := client.Cli.SendMsgToUserByTgid(context.Background(), sendReq)

	if err != nil {
		logger.L.Error("rpcClient.SendMsgToUserByTgid 发送消息失败", zap.String("addr", server.String()), zap.Any("error", err))
		return
	}

	if rsp.GetRetCode() != im_err.OK {
		logger.L.Error("rpcClient.SendMsgToUserByTgid 发送消息响应码错误", zap.String("addr", server.String()), zap.Int32("retCode", rsp.GetRetCode()))
		return
	}

	res = true
	return
}

// 停止用户
func StopUserClient(server *types.Server, uid string) (res bool, err error) {
	client, err := GetClientByAddr(server.String())
	if err != nil {
		return
	}
	sendReq := &rpc.UserIdReq{
		Uid: uid,
	}

	_, err = client.Cli.StopUserClient(context.Background(), sendReq)
	if err != nil {
		logger.L.Error("rpcClient.StopUser 停止用户失败", zap.String("addr", server.String()), zap.Any("error", err))
		return
	}

	res = true
	return
}

// 获取所有系统信息
func GetAllSysInfo() (sysList []*types.SysInfo, err error) {

	lock.RLock()
	defer lock.RUnlock()

	sysList = make([]*types.SysInfo, 0)
	for _, client := range clients {
		sysInfo, err := GetSysInfo(client)
		if err != nil {
			continue
		}
		sysList = append(sysList, sysInfo)
	}
	if len(sysList) == 0 {
		err = fmt.Errorf("rpcClient.GetAllSysInfo 没有获取到服务信息")
		logger.L.Error("rpcClient.GetAllSysInfo 没有获取到服务信息")
		return
	}

	return
}

// 获取系统信息
func GetSysInfo(client *Client) (sysInfo *types.SysInfo, err error) {

	rsp, err := client.Cli.GetSysInfo(context.Background())
	if err != nil {
		logger.L.Error("rpcClient.GetSysInfo 获取系统信息失败", zap.String("addr", client.Addr), zap.Any("error", err))
		return
	}
	sysInfo = &types.SysInfo{}
	err = json.Unmarshal(rsp.GetData(), sysInfo)
	if err != nil {
		logger.L.Error("rpcClient.GetSysInfo 解析系统信息失败", zap.String("addr", client.Addr), zap.Any("error", err))
		return
	}

	return
}
