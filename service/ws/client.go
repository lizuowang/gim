/**
 * Created by GoLand.
 * User: link1st
 * Date: 2019-07-25
 * Time: 16:24
 */

package ws

import (
	"runtime/debug"
	"sync"
	"time"

	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lxzan/gws"
	"go.uber.org/zap"
)

type ClientState int8

const (
	StateOnline  ClientState = 1 // 在线
	StateOffline ClientState = 2
)

// 用户连接
type Client struct {
	lock          sync.RWMutex    // 读写锁
	Socket        *gws.Conn       // 用户连接
	UserId        string          // 用户Id，用户登录以后才有
	RandUuid      string          // 随机uuid
	State         ClientState     // 连接状态
	FirstTime     uint64          // 首次连接时间
	HeartbeatTime uint64          // 用户上次心跳时间
	TGroupLock    sync.RWMutex    // 读写锁
	TGroup        map[string]bool // 临时组
	MsgCode       gws.Opcode      // 消息类型
	sendChan      chan []byte
}

// 初始化
func NewClient(socket *gws.Conn, uid string, randUuid string, msgCode gws.Opcode) (client *Client) {
	cTIme := uint64(time.Now().Unix())
	client = &Client{
		UserId:        uid,
		RandUuid:      randUuid,
		State:         StateOnline,
		Socket:        socket,
		FirstTime:     cTIme,
		HeartbeatTime: cTIme,
		TGroup:        make(map[string]bool),
		MsgCode:       msgCode,
		sendChan:      make(chan []byte, Config.ClientMsgChanMax), // 增加队列容量以应对高并发
	}

	return
}

func (c *Client) InitStart() {
	// 发送消息循环
	go c.SendMsgLoop()

	// 连接回调
	ClientM.OnConnect(c)
	// 设置连接超时
	c.SetConnDeadLine()
}

/**    临时组       */
// 是否在临时组
func (c *Client) IsInTGroup(group_id string) bool {
	c.TGroupLock.RLock()
	defer c.TGroupLock.RUnlock()
	_, ok := c.TGroup[group_id]
	return ok
}

// 进入临时组
func (c *Client) InTGroup(group_id string) {

	TGroupM.UserJoinTGroup(group_id, c.UserId)

	c.TGroupLock.Lock()
	defer c.TGroupLock.Unlock()

	c.TGroup[group_id] = true
}

// 退出临时组
func (c *Client) OutTGroup(group_id string) {

	TGroupM.UserOutTGroup(group_id, c.UserId)

	c.TGroupLock.Lock()
	defer c.TGroupLock.Unlock()

	delete(c.TGroup, group_id)
}

// 退出临时组
func (c *Client) _outTGroup(group_id string) {
	TGroupM.UserOutTGroup(group_id, c.UserId)

	delete(c.TGroup, group_id)
}

// 退出所有临时组
func (c *Client) OutAllTGroup() {
	c.TGroupLock.Lock()
	defer c.TGroupLock.Unlock()

	for group_id := range c.TGroup {
		c._outTGroup(group_id)
	}
}

// GetKey 获取 key
func (c *Client) GetKey() (key string) {
	key = c.UserId
	return
}

/** 消息处理 */

// 发送responseMsg
// sync 1:同步 2:异步
func (c *Client) SendResponseMsg(response *types.Response, sync int8) (err error) {
	headByte, err := response.Bytes()
	if err != nil {
		logger.L.Error("Client.SendResponseMsg error", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		return err
	}
	err = c.SafeSendMsg(headByte, sync)
	return
}

// 发送response
func (c *Client) SendResponseData(rData *types.ResponseData, seq string, cmd string, mstType int) (err error) {
	response := types.NewResponse(seq, cmd, mstType, rData)
	return c.SendResponseMsg(response, 1)
}

// 发送 错误信息
func (c *Client) SendErr(errMsg string, errType int, seq string, cmd string) (err error) {
	a := types.NewErrAData(errMsg, errType)
	rData := types.NewResponseData(a)

	err = c.SendResponseData(rData, seq, cmd, types.MSG_TYPE_RESPONSE)
	return
}

// SendMsg 发送数据
func (c *Client) SendMsg(msg []byte) error {

	return c.SafeSendMsg(msg, 1)
}

// SendMsgAsync 异步发送数据
func (c *Client) SendMsgAsync(msg []byte) error {
	return c.SafeSendMsg(msg, 2)
}

// SafeSendMsg 发送数据
// sendType 1:同步发送 2:异步发送
func (c *Client) SafeSendMsg(msg []byte, sendType int8) error {

	if c.State != StateOnline {
		return im_err.NewImError(im_err.ErrUserOffline)
	}

	//拦截恐慌
	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("Client.SafeSendMsg error", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
			c.Close()
		}
	}()

	if sendType == 1 { // 同步发送
		c.Socket.WriteMessage(c.MsgCode, msg)
	} else {
		select {
		case c.sendChan <- msg:
		default:
			logger.L.Error("Client.SafeSendMsg chan is full", zap.String("uid", c.UserId))
			c.Close()
			return im_err.NewImError(im_err.ErrSendChanFull)
		}
	}
	return nil
}

/** 发送消息循环 */
func (c *Client) SendMsgLoop() {

	defer c.Close()
	for msg := range c.sendChan {
		// 增加错误处理，避免单个消息发送失败影响整个连接
		err := c.Socket.WriteMessage(c.MsgCode, msg)
		if err != nil {
			logger.L.Error("Client.SendMsgLoop write message error",
				zap.String("uid", c.UserId),
				zap.Error(err))
			break
		}
	}
}

/** 关闭连接 */

// close 关闭客户端连接
func (c *Client) Close() {
	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("Client.Close error", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		}
	}()

	c.lock.Lock()
	defer c.lock.Unlock()
	if c.State == StateOffline {
		return
	}
	c.State = StateOffline
	c.Socket.NetConn().Close()
}

// 监听关闭时间
func (c *Client) OnClose() {
	c.State = StateOffline

	// 退组
	c.OutAllTGroup()
	// 删除客户端
	ClientM.DelClient(c.GetKey())
	ClientM.DelClientConn(c)
	//删除在线信息
	DelUserOnlineInfo(c.GetKey())

	//关闭chan
	close(c.sendChan)

	// 回调
	if Config.OnClientClose != nil {
		Config.OnClientClose(c)
	}

}

/** 心跳保活 */

func (c *Client) Heartbeat() {

	ResetUserOnlineExpTime(c.UserId)
	c.SetConnDeadLine()

	// 更新心跳时间
	c.HeartbeatTime = uint64(time.Now().Unix())
}

// 延长连接事件
func (c *Client) SetConnDeadLine() {
	if connKeepTime > 0 {
		c.Socket.SetReadDeadline(time.Now().Add(connKeepTime))
	}
}
