package msg_queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lizuowang/gim/service/ws"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Consumer struct {
	quitChan chan bool
	idx      int
}

var (
	list_key      = "_gim:queue:common:msg:list"
	maxGoroutines = 20 //消费者最大数量
	mu            sync.Mutex
	consumers     = make(map[*Consumer]bool)
	mqConfig      *MqConfig //配置
	freeCNum      int32     //空闲协程数量
	freeTimes     int       //空闲次数
	qCtx          context.Context
	reNum         int
)

// 增加空闲协程数量
func incrFreeCNum() {
	atomic.AddInt32(&freeCNum, 1)
}

// 减少空闲协程数量
func decrFreeCNum() {
	atomic.AddInt32(&freeCNum, -1)
}

// 获取空闲协程数量
func GetFreeCNum() int32 {
	return atomic.LoadInt32(&freeCNum)
}

// 实例化一个消费者
func NewConsumer(conNum int) *Consumer {
	return &Consumer{
		quitChan: make(chan bool),
		idx:      conNum,
	}
}

// 启动消费者
func (c *Consumer) consumeMsg() {
	defer func() {
		mu.Lock()
		delete(consumers, c)
		mu.Unlock()
		logger.L.Info("Consumer.consumeMsg stop ", zap.Int("idx", c.idx), zap.Int("num", len(consumers)))

	}()

	logger.L.Info("Consumer.consumeMsg start ", zap.Int("num", c.idx))

	// 增加空闲协程数量
	incrFreeCNum()

	// 减少空闲协程数量
	defer decrFreeCNum()

	for {
		select {
		case <-c.quitChan:
			return
		default:
			msg, err := mqConfig.RedisClient.BLPop(context.Background(), time.Second*5, list_key).Result()
			// 减少空闲协程数量
			decrFreeCNum()
			if err != nil {
				if err != redis.Nil {
					logger.L.Error("Consumer.consumeMsg error", zap.Int("idx", c.idx), zap.Error(err))
					time.Sleep(time.Second * 1)
				}
			} else if msg != nil {
				c.handleMsg(msg[1])
			}

			// 增加空闲协程数量
			incrFreeCNum()
		}
	}
}

// 处理消息
func (c *Consumer) handleMsg(msg string) {

	subMsg := &types.SubMsg{}

	err := json.Unmarshal([]byte(msg), subMsg)
	if err != nil {
		logger.L.Error("Consumer.handleMsg 接收消息解析错误 ", zap.Any("error", err), zap.String("msg", msg))
		return
	}

	if subMsg.Mol == "" {
		logger.L.Error("Consumer.handleMsg 接收消息信息缺失 ", zap.String("msg", msg))
		return
	}

	var responseData *types.ResponseData

	if subMsg.Ctrl == "" {
		responseData = types.NewMolResponseData(subMsg.Data, subMsg.Mol)
	} else {
		responseData = types.NewCtrlResponseData(subMsg.Data, subMsg.Mol, subMsg.Ctrl)
	}

	if subMsg.Tgid != "" { //向组内发送
		if subMsg.ToUid != "" { // 向组内用户发送
			ws.Proxy.SendResDataToUserByTgid(subMsg.ToUid, subMsg.Tgid, responseData)
		} else { // 向组内发送
			ws.Proxy.SendResDataToGroup(subMsg.Tgid, responseData, subMsg.ExceptUid)
		}
	} else if subMsg.ToUid != "" { //向用户发送
		ws.Proxy.SendResDataToUser(subMsg.ToUid, responseData)
	}

}

// 关闭一个消费者
func (c *Consumer) stop() {
	defer func() {
		if r := recover(); r != nil {
			logger.L.Error("msg.Consumer.stop error ", zap.Int("idx", c.idx), zap.Any("error", r))
		}
	}()
	close(c.quitChan)
}

type MqConfig struct {
	KeyPrefix   string `json:"key_prefix"` // 消息队列key前缀
	RedisClient *redis.Client
}

// 初始化并启动消息队列
func InitStartList(config *MqConfig, ctx context.Context) {
	mqConfig = config
	list_key = mqConfig.KeyPrefix + list_key

	mqConfig.RedisClient.Del(context.Background(), list_key)

	qCtx = ctx
	go startList()
}

// 启动消息队列
func startList() {

	defer func() { //异常退出是 重新启动
		if r := recover(); r != nil {
			logger.L.Error("wsMsgQueue.startList stop error ", zap.Any("error", r))
			reNum++
			if reNum > 10 {
				panic(fmt.Errorf("wsMsgQueue.startList stop error "))
			}
			// 重新启动startList或者其他恢复操作
			time.Sleep(time.Second * 1)
			go startList()
		}
	}()

	for {
		select {
		case <-qCtx.Done():
			logger.L.Info("wsMsgQueue.startList 关闭")
			log.Println("wsMsgQueue.startList close ")
			return
		default:
			sleepTime := manageConsumer()
			time.Sleep(sleepTime) // 每1秒检查一次

			// 重置重试次数
			reNum = 0
		}
	}

}

// 管理消费者
func manageConsumer() time.Duration {
	mu.Lock()
	defer mu.Unlock()
	sleepTime := time.Second * 1

	consumerLength := getConsumeNum()
	if consumerLength < 1 { //默认启动一个协程
		startMultiConsumer(2)
		return sleepTime
	}

	length := GetMsgListLen()
	freeCNum := GetFreeCNum()

	if length >= 2 && consumerLength < maxGoroutines { //启动一个消费协程
		startMultiConsumer(3)
		freeTimes = 0
	} else if length < 1 && consumerLength > 2 && freeCNum > 0 { //关闭一个消费协程
		freeTimes++
		if freeTimes > 60 {
			stopConsumer()
			sleepTime = time.Second * 5
			freeTimes = 0
		}
	} else {
		freeTimes = 0
	}

	return sleepTime
}

// 启动多个消费者
func startMultiConsumer(num int) {
	for i := 0; i < num; i++ {
		consumer := NewConsumer(len(consumers) + 1)
		consumers[consumer] = true
		go consumer.consumeMsg()
	}
}

// 获取消费者数量
func getConsumeNum() int {
	return len(consumers)
}

// 关闭一个消费者
func stopConsumer() {
	for consumer := range consumers {
		consumer.stop()
		break
	}
}

// 消息堵塞
func GetMsgListLen() int64 {
	length, err := mqConfig.RedisClient.LLen(context.Background(), list_key).Result()
	if err != nil {
		logger.L.Error("GetMsgListLen error", zap.Error(err))
		return 0
	}
	return length
}

// 消费者数量
func GetConsumeLen() int {
	return len(consumers)
}

func AddTestMsg() {
	return
	num := 10000

	for i := 0; i < num; i++ {
		now := time.Now().UnixNano() / int64(time.Millisecond)
		msg := fmt.Sprintf("{\"data\":{\"id\":%d},\"mol\":\"a\",\"ctrl\":\"c\",\"tgid\":\"\",\"except_uid\":\"\",\"to_uid\":\"\", \"c_time\":%d}", i, now)
		go addMsg(msg)
	}

}

func addMsg(msg string) {
	start := time.Now()
	mqConfig.RedisClient.RPush(context.Background(), list_key, msg)
	elapsed := time.Since(start)
	logger.L.Info("AddTestMsg execution time", zap.Duration("elapsed", elapsed))
}

// func AddTestMsg() {
// 	num := 20000

// 	key := "ws:queue:tg_act_7160_sid_6_hid_2358228"
// 	for i := 0; i < num; i++ {
// 		now := time.Now().UnixNano() / int64(time.Millisecond)
// 		msg := fmt.Sprintf("{\"data\":{\"id\":%d},\"mol\":\"a\",\"ctrl\":\"c\",\"tgid\":\"\",\"except_uid\":\"\",\"to_uid\":\"\", \"c_time\":%d}", i, now)
// 		go redisClient.Publish(context.Background(), key, msg)
// 	}

// }
