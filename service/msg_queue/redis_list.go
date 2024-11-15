package msg_queue

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lizuowang/gim/service/ws"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Consumer struct {
	quitChan chan bool
	idx      int64
}

var (
	list_key            = "_gim:queue:common:msg:list"
	maxGoroutines       = 20 //消费者最大数量
	consumeNum    int64 = 0
	mu            sync.Mutex
	consumers     = make(map[*Consumer]bool)
	mqConfig      *MqConfig //配置
)

// 实例化一个消费者
func NewConsumer(conNum int64) *Consumer {
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
		logger.L.Info("Consumer.consumeMsg stop ", zap.Int64("idx", c.idx), zap.Int64("num", c.idx))

	}()

	logger.L.Info("Consumer.consumeMsg start ", zap.Int64("num", c.idx))

	for {
		select {
		case <-c.quitChan:
			return
		default:
			msg, err := mqConfig.RedisClient.BLPop(context.Background(), time.Second*5, list_key).Result()
			if err != nil {
				if err != redis.Nil {
					logger.L.Error("Consumer.consumeMsg error", zap.Int64("idx", c.idx), zap.Error(err))
					time.Sleep(time.Second * 1)
				}
			} else if msg != nil {
				c.handleMsg(msg[1])
			}

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
			logger.L.Error("msg.Consumer.stop error ", zap.Int64("idx", c.idx), zap.Any("error", r))
		}
	}()
	close(c.quitChan)
}

type MqConfig struct {
	KeyPrefix   string `json:"key_prefix"` // 消息队列key前缀
	RedisClient *redis.Client
}

// 初始化并启动消息队列
func InitStartList(config *MqConfig) {
	mqConfig = config
	list_key = mqConfig.KeyPrefix + list_key

	mqConfig.RedisClient.Del(context.Background(), list_key)

	go startList()
}

// 启动消息队列
func startList() {

	defer func() { //异常退出是 重新启动
		if r := recover(); r != nil {
			logger.L.Error("msg.startList stop error ", zap.Any("error", r))
			// 重新启动startList或者其他恢复操作
			go startList()
		}
	}()

	for {
		sleepTime := manageConsumer()
		time.Sleep(sleepTime) // 每1秒检查一次
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

	if length >= 1 && consumerLength < maxGoroutines { //启动一个消费协程
		startMultiConsumer(3)
	} else if length < 1 && consumerLength > 2 { //关闭一个消费协程
		stopConsumer()
		sleepTime = time.Second * 5
	}

	return sleepTime
}

// 启动多个消费者
func startMultiConsumer(num int) {
	for i := 0; i < num; i++ {
		consumeNum++
		consumer := NewConsumer(consumeNum)
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
