package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"time"

	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	rpcClient "github.com/lizuowang/gim/service/api/client"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	SUB_TYPE_GROUP = 1                     //消息类型 组消息
	SUB_GROUP_KEY  = "_gim:sub:trgoup:key" //订阅key
)

var (
	subMsgConsumerMax = 20
	subMsgConsumerMin = 1
	subCtx            context.Context
	redisProxy        *RedisProxy
)

type RedisProxy struct {
	LocalProxy
	RedisClient *redis.Client
	ConsumerGm  map[*subMsgConsumer]bool
	MsgCh       <-chan *redis.Message
	Lock        sync.RWMutex
	freeTimes   int // 空闲次数
}

type subMsgConsumer struct {
	Ctx    context.Context
	Cancel context.CancelFunc
	Idx    int
}

// 实例化
func NewRedisProxy(redisClient *redis.Client) *RedisProxy {
	redisProxy = &RedisProxy{
		RedisClient: redisClient,
		ConsumerGm:  make(map[*subMsgConsumer]bool),
	}
	go redisProxy.SubV2()
	return redisProxy
}

// 订阅
func (rp *RedisProxy) SubV2() {
	// 订阅key
	subKey := rp.GetSubKey()
	subCtx = context.Background()
	pubsub := rp.RedisClient.Subscribe(subCtx, subKey)
	pubsub.ReceiveTimeout(subCtx, 0)

	rp.MsgCh = pubsub.Channel()

	defer func() {

		logger.L.Error("redisProxy.Sub 异常退出 ")
		panic(fmt.Errorf("redsi proxy sub error "))

	}()
	defer pubsub.Close()

	log.Println("redis proxy start success ")

	rp.AddConsumer()

	// 管理消费
	for {
		sTime := rp.consumerManage()
		time.Sleep(sTime)
	}

}

// 消费者管理
func (rp *RedisProxy) consumerManage() time.Duration {
	rp.Lock.Lock()
	defer rp.Lock.Unlock()

	consumerLen := len(rp.ConsumerGm)
	chLen := len(rp.MsgCh)
	//消息队列长度 大于5 添加一个消费者
	if consumerLen < subMsgConsumerMax && chLen >= 5 {
		rp.AddConsumer()
		rp.freeTimes = 0
	} else if consumerLen == 0 { //如果没有消费者 则启动一个消费者
		rp.AddConsumer()
		rp.freeTimes = 0

	} else if consumerLen > subMsgConsumerMin && chLen == 0 { // 没有消息 减少一个消费者
		rp.freeTimes++
		if rp.freeTimes > 30 {
			for smc := range rp.ConsumerGm {
				smc.Stop()
				break
			}
			rp.freeTimes = 0
		}
	} else {
		rp.freeTimes = 0
	}

	return time.Second
}

// 启动一个消息消费者
func NewSubMsgConsumer(idx int) *subMsgConsumer {
	ctx, Cancel := context.WithCancel(context.Background())
	return &subMsgConsumer{
		Ctx:    ctx,
		Cancel: Cancel,
		Idx:    idx,
	}
}

// 停止消费者
func (smc *subMsgConsumer) Stop() {
	smc.Cancel()
}

func (rp *RedisProxy) AddConsumer() (err error) {

	conNum := len(rp.ConsumerGm)
	if conNum >= subMsgConsumerMax {
		err = fmt.Errorf("subMsgConsumerMax is full ")
		return
	}
	smc := NewSubMsgConsumer(conNum)
	rp.ConsumerGm[smc] = true
	go rp.StartSub(smc)

	return
}

// 开始消费者
func (rp *RedisProxy) StartSub(smc *subMsgConsumer) (err error) {
	defer func() {
		if err != nil {
			logger.L.Error("redisProxy.subMsgConsumer.Sub 消费者异常退出 ", zap.Any("error", err))
		}
		rp.Lock.Lock()
		delete(rp.ConsumerGm, smc)
		logger.L.Info("redisProxy.subMsgConsumer.StartSub  stop ", zap.Int("idx", smc.Idx), zap.Int("num", len(rp.ConsumerGm)))
		rp.Lock.Unlock()
	}()

	logger.L.Info("redisProxy.subMsgConsumer.StartSub start ", zap.Int("idx", smc.Idx), zap.Int("ch_len", len(rp.MsgCh)))

	for {
		select {
		case <-smc.Ctx.Done(): //监听是否退出
			return
		case msg, ok := <-rp.MsgCh: //监听消息
			if !ok {
				return
			}
			// 处理msg
			rp.handleMsg(msg)
		}
	}
}

// 处理消息
func (rp *RedisProxy) handleMsg(msg *redis.Message) {
	// 解析msg
	subMsg := &SubMsg{}
	err := json.Unmarshal([]byte(msg.Payload), subMsg)
	if err != nil {
		logger.L.Error("redisProxy.handleMsg 接收消息解析错误 ", zap.Any("error", err), zap.String("msg", msg.Payload))
		return
	}

	switch subMsg.SubType {
	case SUB_TYPE_GROUP:
		subGroupMsg := subMsg.Msg
		err = rp.SendMsgToLocalGroup(subGroupMsg.Gid, subGroupMsg.GroupMsg)
		if err != nil {
			logger.L.Error("redisProxy.handleMsg SendMessageToTGroup error ", zap.Any("error", err))
			return
		}
	default:
		logger.L.Error("redisProxy.handleMsg subMsg type ", zap.Uint8("type", subMsg.SubType))
		return
	}
}

// 定于类型
type SubMsg struct {
	SubType uint8        `json:"subType"` //消息类型
	Msg     *SubGroupMsg `json:"msg"`     //消息
}

// 生成一个订阅消息
func NewSubMsg(msg *SubGroupMsg, subType uint8) *SubMsg {
	usbMsg := &SubMsg{
		SubType: subType,
		Msg:     msg,
	}
	return usbMsg
}

// 订阅组消息
type SubGroupMsg struct {
	Gid      string          `json:"gid"`      //临时组id
	GroupMsg *types.GroupMsg `json:"groupMsg"` //消息
}

// 生成一个订阅组消息
func NewSubGroupMsg(gid string, groupMsg *types.GroupMsg) *SubGroupMsg {
	subGroupMsg := &SubGroupMsg{
		Gid:      gid,
		GroupMsg: groupMsg,
	}
	return subGroupMsg
}

/** 组消息 */

// 向组内发送消息
func (rp *RedisProxy) SendResDataToGroup(gid string, resData *types.ResponseData, exceptUid string) (err error) {
	groupMsg, err := types.GetResDataGroupMsg(resData, exceptUid)
	if err != nil {
		return
	}
	err = rp.SendMsgToGroup(gid, groupMsg)
	return
}

// 向临时组内发送消
func (rp *RedisProxy) SendMsgToGroup(gid string, groupMsg *types.GroupMsg) (err error) {
	subGroupMsg := NewSubGroupMsg(gid, groupMsg)

	subMsg := NewSubMsg(subGroupMsg, SUB_TYPE_GROUP)

	// 将subMsg 转为byte
	subMsgByte, err := json.Marshal(subMsg)
	if err != nil {
		logger.L.Error("redisProxy.SendMsgToGroup json.Marshal error", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		return
	}

	// 发布订阅消息
	err = rp.RedisClient.Publish(context.Background(), rp.GetSubKey(), subMsgByte).Err()
	if err != nil {
		logger.L.Error("redisProxy.SendMsgToGroup publish error ", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		return
	}

	return
}

// 订阅key
func (rp *RedisProxy) GetSubKey() string {
	return Config.RedisPrefix + SUB_GROUP_KEY
}

// 向用户发消息
func (rp *RedisProxy) SendResDataToUser(uid string, resData *types.ResponseData) (err error) {
	if ClientM.HasUser(uid) {
		return rp.SendResDataToLocalUser(uid, resData)
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}
	response := types.NewResponse("", "", types.MSG_TYPE_U2U, resData)
	msgBytes, err := response.Bytes()
	if err != nil {
		return
	}
	_, err = rpcClient.SendMsgToUser(server, uid, msgBytes)

	return
}

// 向组内用户发送消息
func (rp *RedisProxy) SendResDataToUserByTgid(uid string, tgid string, resData *types.ResponseData) (err error) {
	if ClientM.HasUser(uid) {
		return rp.SendResDataToLocalUserByTgid(uid, tgid, resData)
	}

	response := types.NewResponse("", "", types.MSG_TYPE_U2U, resData)
	msgBytes, err := response.Bytes()
	if err != nil {
		return
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}

	_, err = rpcClient.SendMsgToUserByTgid(server, uid, tgid, msgBytes)

	return
}

// 向用户发送消息
func (rp *RedisProxy) SendMsgToUser(uid string, msg []byte) (err error) {
	if ClientM.HasUser(uid) {
		return rp.SendMsgToLocalUser(uid, msg)
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}
	_, err = rpcClient.SendMsgToUser(server, uid, msg)

	return
}

// 向组内用户发消息
func (rp *RedisProxy) SendMsgToUserByTgid(uid string, tgid string, sendMsg []byte) (err error) {
	if ClientM.HasUser(uid) {
		return ClientM.SendMsgToUserByTgid(uid, tgid, sendMsg)
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}
	_, err = rpcClient.SendMsgToUserByTgid(server, uid, tgid, sendMsg)

	return
}

// 停止用户client
func (rp *RedisProxy) StopUserClient(userOnline *UserOnline) {
	//检查是否是本机
	if userOnline.UserIsLocal() {
		rp.StopLocalUserClient(userOnline.UserId)
		return
	}
	if userOnline.RpcPort == "" {
		return
	}
	// 远程stop
	server := types.NewServer(userOnline.AccIp, userOnline.RpcPort)
	rpcClient.StopUserClient(server, userOnline.UserId)
}
