# GIM

## 项目介绍
- 支持集群模式
- 支持消息群发
- msg_queue服务 其他服务 可以发送消息到消息队列 消息队列服务会根据消息类型 转发到对应的客户端、群组
- 可使用ws.Proxy 发送消息（支持集群模式）
```go
type RunProxy interface {
	//发送响应数据
	SendResDataToUser(uid string, resData *types.ResponseData) (err error)                         // 向用户发送消息
	SendResDataToLocalUser(uid string, resData *types.ResponseData) (err error)                    // 向本地用户发送消息
	SendResDataToUserByTgid(uid string, tgid string, resData *types.ResponseData) (err error)      // 向组内用户发送消息
	SendResDataToLocalUserByTgid(uid string, tgid string, resData *types.ResponseData) (err error) // 向本地组内用户发送消息
	SendResDataToGroup(gid string, resData *types.ResponseData, exceptUid string) (err error)      // 向组内发送消息
	SendResDataToLocalGroup(gid string, resData *types.ResponseData, exceptUid string) (err error) // 向本地组内发送消息

	//发送[]byte消息
	SendMsgToLocalGroup(gid string, groupMsg *types.GroupMsg) (err error)     // 向本地组内发送消息
	SendMsgToUser(uid string, msg []byte) (err error)                         // 向用户发送消息
	SendMsgToLocalUser(uid string, msg []byte) (err error)                    // 向本地用户发送消息
	SendMsgToUserByTgid(uid string, tgid string, msg []byte) (err error)      // 向组内用户发送消息
	SendMsgToLocalUserByTgid(uid string, tgid string, msg []byte) (err error) // 向本地组内用户发送消息

	StopLocalUserClient(uid string)        //停止本地用户client
	StopUserClient(userOnline *UserOnline) //停止用户client
}
```

## 项目结构

- pkg 公共包
    - logger 日志
    - etcd_registry etcd注册中心
    - im_err 错误码
    - logger 日志
    - redis_client redis客户端
    - etcd_client etcd客户端
- service 服务
    - api //RPC通信服务
    - ws //websocket服务
    - msg_queue //消息队列服务
    - sys //系统服务
- examples 示例
- server.go 启动服务入口

## 项目依赖

### 初始化日志
```go
logConfig := &logger.LogConfig{
    Level: "debug",//日志级别
    FileName:   "gim.log",//日志文件名
    FilePath:   "./examples/local/server/logs/",
    MaxSize:    100,
    MaxAge:     10,
    MaxBackups: 10,
    Compress:   false,
}
gim.InitLogger(logConfig)

logger.L.Info("这个是info日志示例")
logger.L.Error("这个是error日志示例")
```

### 初始化redis
```go
redisConf := &redis.Options{
    Addr:         "192.168.1.191:1099",
    Password:     "123456",
    DB:           0,
    PoolSize:     10,//连接池大小
    MinIdleConns: 10,//最小空闲连接数
}
redis_client.CreateClient(redisConf)
redisClient := redis_client.GetClient()
```

### 初始化etcd (集群模式时强制依赖etcd)
```go
etcdConf := &clientv3.Config{
    Endpoints:   []string{"192.168.1.191:2379"},//etcd地址
    DialTimeout: 5 * time.Second,//连接超时时间
}
etcd_client.CreateClient(etcdConf)
etcd := etcd_client.GetClient()
```

## 快速开始

### 添加路由
```go
http.HandleFunc("/acc", ws.HandlerUp) // 建立连接
http.HandleFunc("/", gimHttp.HomePage) // 首页
http.HandleFunc("/sysinfo", gimHttp.GetSysInfo) // 系统信息
```

### 启动服务
```go
// 启动gim服务
gim.InitServer(wsConf)

server := &http.Server{
    Addr: fmt.Sprintf("%s:8080", ip),//地址
}


// 启动http服务
go func() {
    if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        log.Fatalf("Could not listen on %s: %v\n", server.Addr, err)
    }
}()

```

### 关闭服务
```go
//关闭服务
defer gim.CloseServer()
```

### 监听信号
```go
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
```

## wsConf 配置

```go
wsConf := &ws.WsConfig{
    // 运行模式
    RunMode: ws.RunModeRedis,
    // redis前缀
    RedisPrefix: "test",
    // rpc端口
    RpcPort: "8091",
    // ws端口
    WsPort: "8080",
}

type WsConfig struct {
	RunMode     RunMode           //运行模式
	RedisPrefix string            //redis前缀
	RpcPort     string            //rpc端口
	WsPort      string            //ws端口
	KeepTime    int               // 用户连接超时时间 0为不限制
	Handler     gws.Event         // 事件处理
	Option      *gws.ServerOption //gws库 的服务配置
	RedisClient *redis.Client     //redis客户端
	EtcdClient  *clientv3.Client  //etcd客户端 集群模式时强制依赖etcd


	OnClientClose func(client *Client) // 用户连接关闭回调
	Version       string               // 版本

}

//OnClientClose func(client *Client) // 用户连接关闭回调
//定义 用户关闭时回调
func OnClientClose(client *Client) {
    log.Println("用户连接关闭", client.Uid)
}

```

###  处理事件 Handler gws.Event
```go
//自定义处理事假 继承ws.Handler
type Handler struct {
	ws.Handler
}

// 添加消息处理方法
handler := &Handler{}
handler.HandleMsg = handleMsg
//handleMsg(client *ws.Client, request *types.Request) (aData types.AData, err error)

```

### Handler 监听事件 可重写方法
```go
// 连接建立
func (h *Handler) OnOpen(socket *gws.Conn) 


``` 

### gws.ServerOption 握手认证
```go
// 握手认证
Authorize: func(r *http.Request, session gws.SessionStorage) bool {
   
    // 你的握手认证逻辑

    // 必须有个uid 全局唯一 表示一个用户 （同一个uid同时只能创建一个连接 重复创建会关闭之前和当前连接 ）
    uidStr := r.URL.Query().Get("uid")

    if uidStr == "" {
        return false
    }
    session.Store("uid", uidStr)
    return true
}

```

### client 客户端
```go
// 获取客户端
client := ws.ClientM.GetClient(uid)
// 进入组
client.InTGroup("group_id")
// 退出组
client.OutTGroup("group_id")
```


