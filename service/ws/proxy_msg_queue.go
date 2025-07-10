package ws

import (
	"context"
	"encoding/json"
	"runtime/debug"
	"time"

	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lizuowang/gmq/task_worker"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ProxyMsgQueue struct {
	WorkerM   *task_worker.WorkerM
	GlobalKey string
	LocalKey  string
}

var (
	msgChanMaxLen  = 1000            // 消息最大长度
	msgPushTimeout = 5 * time.Second // 5秒
)

// 处理订阅消息
func handleSubMsg(msg string) (newMsg string) {

	subMsg := &types.SubMsg{}
	err := json.Unmarshal([]byte(msg), subMsg)
	if err != nil {
		//打印调用栈
		logger.L.Error("proxy.handleSubMsg 接收消息解析错误 ", zap.Any("error", err), zap.String("msg", msg), zap.String("debug_stack", string(debug.Stack())))
		return
	}

	var responseData *types.ResponseData
	if subMsg.Mol == "" {
		responseData = types.NewResponseData(subMsg.Data)
	} else if subMsg.Ctrl == "" {
		responseData = types.NewMolResponseData(subMsg.Data, subMsg.Mol)
	} else {
		responseData = types.NewCtrlResponseData(subMsg.Data, subMsg.Mol, subMsg.Ctrl)
	}

	if subMsg.Tgid != "" { //向组内发送
		if subMsg.ToUid != "" { // 向组内用户发送
			Proxy.SendResDataToUserByTgid(subMsg.ToUid, subMsg.Tgid, responseData)
		} else { // 向本地组内发送
			Proxy.SendResDataToLocalGroup(subMsg.Tgid, responseData, subMsg.ExceptUid)
		}
	} else if subMsg.ToUid != "" { //向用户发送
		Proxy.SendResDataToUser(subMsg.ToUid, responseData)
	}

	return
}

func NewProxyMsgQueue(ctx context.Context) *ProxyMsgQueue {

	// 初始化订阅消息的协程
	WMConf := &task_worker.WorkerMConf{
		Handler:   handleSubMsg,
		MinWorker: 2,
		MaxWorker: 100,
		AddWorker: 10,
		WaitNum:   2,
		FreeTimes: 10,
		L:         logger.L,
		Name:      "imMsgQueue",
		ChanSize:  msgChanMaxLen,
	}
	proxyMsgQueue := &ProxyMsgQueue{
		WorkerM:   task_worker.NewWorkerM(WMConf),
		GlobalKey: Config.RedisPrefix + "_gim:sub:global",
		LocalKey:  Config.RedisPrefix + "_gim:sub:local:" + serverIp + ":" + Config.WsPort,
	}

	proxyMsgQueue.subMsg(Config.RedisClient, ctx)

	return proxyMsgQueue
}

// 发布全局消息
func (mq *ProxyMsgQueue) PublishGlobalMsg(subMsg *types.SubMsg) error {
	subMsgByte, err := json.Marshal(subMsg)
	if err != nil {
		logger.L.Error("proxy.PublishGlobalMsg json.Marshal error", zap.Any("error", err), zap.Any("subMsg", subMsg))
		return err
	}

	return Config.RedisClient.Publish(context.Background(), mq.GlobalKey, subMsgByte).Err()
}

// 订阅消息
func (mq *ProxyMsgQueue) subMsg(redisClient *redis.Client, ctx context.Context) {

	pubsub := redisClient.Subscribe(ctx, mq.GlobalKey, mq.LocalKey)
	pubsub.ReceiveTimeout(ctx, 0)

	go mq.readSubMsg(pubsub, ctx)
}

// 读取订阅消息
func (mq *ProxyMsgQueue) readSubMsg(pubSub *redis.PubSub, ctx context.Context) {

	timer := time.NewTimer(time.Minute)
	timer.Stop()

	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("proxy.readSubMsg error: ", zap.Any("error", err))
		}
		logger.L.Info("proxy.readSubMsg close")
	}()
	defer pubSub.Close()

	logger.L.Info("proxy.readSubMsg start")

	var errCount int
	for {

		select {
		case <-ctx.Done():
			return
		default:
		}

		msg, err := pubSub.ReceiveMessage(ctx)
		if err != nil {
			if err == redis.ErrClosed {
				return
			}
			if errCount > 0 {
				time.Sleep(100 * time.Millisecond)
			}
			errCount++
			continue
		}

		errCount = 0

		// 如果消息队列满了，则使用超时发送
		if len(mq.WorkerM.MsgChan) >= msgChanMaxLen {
			// 重置定时器
			timer.Reset(msgPushTimeout)
			mq.pubMsgWithTimeout(msg.Payload, timer)
		} else {
			// 直接发送
			mq.WorkerM.PushMsg(msg.Payload)
		}

	}

}

// 有超时的发送消息
func (mq *ProxyMsgQueue) pubMsgWithTimeout(msg string, timer *time.Timer) {
	select {
	case mq.WorkerM.MsgChan <- msg:
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		logger.L.Error("proxy.pubMsgWithTimeout error: channel is full for  (message is dropped)", zap.String("message", msg))
	}
}

func (mq *ProxyMsgQueue) rePushMsg(msg string) {

}

func (mq *ProxyMsgQueue) StopWorkerM() {
	if mq.WorkerM != nil {
		close(mq.WorkerM.MsgChan)
		mq.WorkerM.Stop(mq.rePushMsg)
	}
}
