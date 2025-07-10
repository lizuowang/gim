package gim

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/cloudwego/kitex/pkg/klog"
	"github.com/lizuowang/gim/pkg/etcd_client"
	"github.com/lizuowang/gim/pkg/logger"
	rpcClient "github.com/lizuowang/gim/service/api/client"
	rpcServer "github.com/lizuowang/gim/service/api/server"
	"github.com/lizuowang/gim/service/ws"
	"go.uber.org/zap"
)

var (
	IsInitLogger bool
	runMode      ws.RunMode
	rpcFile      *os.File
	serCtx       context.Context
	serCancel    context.CancelFunc
)

func InitServer(config *ws.WsConfig) error {
	if !IsInitLogger {
		panic("请先初始化日志")
	}
	serCtx, serCancel = context.WithCancel(context.Background())
	ws.InitWs(config, serCtx)

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

	// mqConfig := &msg_queue.MqConfig{
	// 	RedisClient: config.RedisClient,
	// 	KeyPrefix:   config.RedisPrefix,
	// }
	// msg_queue.InitStartList(mqConfig, serCtx)

	return nil
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
	serCancel()
	ws.CloseServer()
	if runMode == ws.RunModeRedis {
		rpcServer.OnShutdown()
	}

	//关闭etcd
	etcd_client.Close()

	//关闭日志系统
	logger.OnClose()

	rpcFile.Close()

	//休眠2秒等待服务全部关闭
	time.Sleep(2 * time.Second)
}
