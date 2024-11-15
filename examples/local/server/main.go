package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/lizuowang/gim"
	"github.com/lizuowang/gim/pkg/etcd_client"
	"github.com/lizuowang/gim/pkg/helper"
	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/redis_client"
	"github.com/lizuowang/gim/pkg/types"
	gimHttp "github.com/lizuowang/gim/service/http"
	"github.com/lizuowang/gim/service/ws"
	"github.com/lxzan/gws"
	"github.com/redis/go-redis/v9"
	clientv3 "go.etcd.io/etcd/client/v3"
)

func main() {

	// 初始化日志
	logConfig := &logger.LogConfig{
		Level: "debug",
		//FileName:   "/data/server/lizi/gim/examples/local/server/logs/server.log",
		FileName:   "gim.log",
		FilePath:   "./examples/local/server/logs/",
		MaxSize:    100,
		MaxAge:     10,
		MaxBackups: 10,
		Compress:   false,
	}
	gim.InitLogger(logConfig)

	redisConf := &redis.Options{
		Addr:         "192.168.1.191:1099",
		Password:     "123456",
		DB:           0,
		PoolSize:     10,
		MinIdleConns: 10,
	}
	redis_client.CreateClient(redisConf)
	redisClient := redis_client.GetClient()

	gwsOption := &gws.ServerOption{
		CheckUtf8Enabled: true,
		Recovery:         gws.Recovery,
		PermessageDeflate: gws.PermessageDeflate{
			Enabled:               false,
			ServerContextTakeover: true,
			ClientContextTakeover: true,
		},

		Authorize: func(r *http.Request, session gws.SessionStorage) bool {
			//全局自增一个 uid
			// uid := redisClient.Incr(context.Background(), "test:uid").Val()
			//转为字符串
			// uidStr := strconv.FormatInt(uid, 10)

			uidStr := r.URL.Query().Get("uid")

			if uidStr == "" {
				return false
			}
			session.Store("uid", uidStr)
			return true
		},
	}

	etcdConf := &clientv3.Config{
		Endpoints:   []string{"192.168.1.191:2379"},
		DialTimeout: 5 * time.Second,
	}
	etcd_client.CreateClient(etcdConf)
	etcd := etcd_client.GetClient()

	handler := &Handler{}
	handler.HandleMsg = handleMsg
	wsConf := &ws.WsConfig{
		RunMode:     ws.RunModeRedis,
		RedisPrefix: "test",
		WsPort:      "8080",
		RpcPort:     "8091",
		Handler:     handler,
		RedisClient: redisClient,
		Option:      gwsOption,
		EtcdClient:  etcd,
	}
	gim.InitServer(wsConf)

	// Create a new HTTP server
	ip := helper.GetServerIp()
	server := &http.Server{
		Addr: fmt.Sprintf("%s:8080", ip),
	}

	http.HandleFunc("/acc", ws.HandlerUp)
	http.HandleFunc("/", gimHttp.HomePage)
	http.HandleFunc("/sysinfo", gimHttp.GetSysInfo)
	log.Println("server start", server.Addr)
	// Start the server in a goroutine
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
		}
	}()

	defer gim.CloseServer()

	// Create a channel to listen for interrupt or terminate signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt)

	// Block until a signal is received
	<-stop

	// Create a deadline to wait for
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Attempt to gracefully shutdown the server
	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exiting")

}

type Handler struct {
	ws.Handler
}

func (h *Handler) OnMessage(socket *gws.Conn, message *gws.Message) {
	defer message.Close()
	// client, ok := ws.ClientM.GetClientByConn(socket)
	// if ok {
	// 	client.SendMsg(message.Bytes())
	// }

	// resp := types.NewResponse("123", "getName", 1, &types.ResponseData{
	// 	S: 1,
	// 	A: types.AData{"tttt": "ddddd"},
	// })
	// ws.TGroupM.SendMessageToTGroup("ccc", &types.GroupMsg{
	// 	Response: resp,
	// })

	client, ok := ws.ClientM.GetClientByConn(socket)
	if ok {
		client.InTGroup("ccc")
		// client.SendResponseData(resData, "101000001", "101000001", types.MSG_TYPE_RESPONSE)
	}

	resData := &types.ResponseData{
		S: 1,
		A: types.AData{"tttt": "私聊"},
	}

	ws.Proxy.SendResDataToUserByTgid("123", "ccc", resData)

	resData = &types.ResponseData{
		S: 1,
		A: types.AData{"tttt": "广播"},
	}

	ws.Proxy.SendResDataToGroup("ccc", resData, "101000001")

}

func (h *Handler) OnOpen(socket *gws.Conn) {

}

func handleMsg(client *ws.Client, request *types.Request) (aData types.AData, err error) {

	aData = types.AData{"tttt": "响应"}
	return
}
