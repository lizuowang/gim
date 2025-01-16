package gim

import (
	"fmt"
	"log"
	"os"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/lizuowang/gim/pkg/etcd_client"
	"github.com/lizuowang/gim/pkg/etcd_registry"
	"github.com/lizuowang/gim/pkg/logger"
	rpcClient "github.com/lizuowang/gim/service/api/client"
	rpcServer "github.com/lizuowang/gim/service/api/server"
	"github.com/lizuowang/gim/service/msg_queue"
	"github.com/lizuowang/gim/service/ws"
	"go.uber.org/zap"
)

var (
	IsInitLogger bool
	etcdRegistry *etcd_registry.ServiceRegister
	runMode      ws.RunMode
	rpcFile      *os.File
)

func InitServer(config *ws.WsConfig) {
	if !IsInitLogger {
		panic("请先初始化日志")
	}
	ws.InitWs(config)

	runMode = config.RunMode
	if config.RunMode == ws.RunModeRedis {
		if config.EtcdClient == nil {
			panic("没有etcd客户端")
		}
		if config.RpcPort == "" {
			panic("rpc端口不能为空")
		}
		serverName := fmt.Sprintf("%s/%s/", config.RedisPrefix, "rpc")
		addr := fmt.Sprintf("%s:%s", ws.GetServerIp(), config.RpcPort)
		// 注册服务
		rpcServer.InitServer(config.EtcdClient, serverName, addr)

		// 发现rpc服务
		rpcClient.InitClient(serverName, config.EtcdClient)

	}

	mqConfig := &msg_queue.MqConfig{
		RedisClient: config.RedisClient,
		KeyPrefix:   config.RedisPrefix,
	}
	msg_queue.InitStartList(mqConfig)
}

// 初始化日志
func InitLogger(config *logger.LogConfig) *zap.Logger {
	err := logger.InitLogger(config)
	if err != nil {
		panic(fmt.Errorf("初始化日志失败: %s ", err))
	}

	logger.L.Info("开始 日志初始化")

	IsInitLogger = true

	InitKLog(config)

	return logger.L
}

// 初始化klog
func InitKLog(config *logger.LogConfig) {
	// 路径不存在时 创建路径
	if _, err := os.Stat(config.FilePath); os.IsNotExist(err) {
		os.MkdirAll(config.FilePath, os.ModePerm)
	}

	rpcFile, err := os.OpenFile(config.FilePath+"rpc.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Println("打开日志文件失败", err)
	}

	klog.SetOutput(rpcFile)
}

// CloseServer 关闭服务
func CloseServer() {
	log.Println("关闭gim服务")
	ws.CloseServer()
	if runMode == ws.RunModeRedis {
		rpcServer.OnShutdown()
	}

	//关闭etcd
	etcd_client.Close()

	//关闭日志系统
	logger.OnClose()

	rpcFile.Close()
}
