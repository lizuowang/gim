package ws

import (
	"context"
	"fmt"
	"time"

	"github.com/lizuowang/gim/pkg/helper"
	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lxzan/gws"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type RunMode uint8

const (
	RunModeLocal RunMode = iota // 本地模式
	RunModeRedis                // 集群模式

	DefaultMsgChanMax = 500 // 消息通道最大长度
)

type WsConfig struct {
	RunMode          RunMode           //运行模式
	RedisPrefix      string            //redis前缀
	RpcPort          string            //rpc端口
	WsPort           string            //ws端口
	KeepTime         int               // 用户连接超时时间 0为不限制
	Handler          gws.Event         // 事件处理
	Option           *gws.ServerOption //服务配置
	RedisClient      *redis.Client     //redis客户端
	EtcdClient       *clientv3.Client  //etcd客户端 集群模式时强制依赖etcd
	ClientMsgChanMax int               // 客户端消息通道最大长度
	GroupMsgChanMax  int               // 组消息通道最大长度

	OnClientClose func(client *Client) // 用户连接关闭回调
	Version       string               // 版本

}

var (
	Config       *WsConfig
	upgrader     *gws.Upgrader
	connKeepTime = 0 * time.Second // 连接超时时间
	serverIp     string

	ClientM    *ClientManager // 客户端管理器
	TGroupM    *TGroupManager // 临时组管理器
	ServerName string
)

func InitWs(c *WsConfig, ctx context.Context) {

	if c.Option == nil {
		panic("ws option is nil")
	}
	if c.Option.Logger == nil {
		c.Option.Logger = im_err.NewErrLog()
	}
	// 设置消息编码
	// if c.MsgCode == 0 {
	// 	c.MsgCode = gws.OpcodeBinary
	// }
	if c.ClientMsgChanMax <= 0 {
		c.ClientMsgChanMax = DefaultMsgChanMax
	}
	if c.GroupMsgChanMax <= 0 {
		c.GroupMsgChanMax = DefaultMsgChanMax
	}

	serverIp = helper.GetServerIp()
	Config = c
	upgrader = gws.NewUpgrader(c.Handler, c.Option)

	// 初始化管理器
	ClientM = NewClientManager()
	TGroupM = NewTGroupManager()

	if Config.KeepTime > 0 {
		connKeepTime = time.Duration(Config.KeepTime) * time.Second
	}

	//获取ip的最后一段
	ServerName = fmt.Sprintf("%s:%s", helper.GetLastSegmentOfIP(serverIp), Config.WsPort)

	//初始化代理
	InitProxy(ctx)
}

// GetServerIp 获取服务ip
func GetServerIp() string {
	return serverIp
}

// CloseServer 关闭服务
func CloseServer() {
	stopProxyWM()
}
