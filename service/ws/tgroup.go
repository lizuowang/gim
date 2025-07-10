package ws

import (
	"runtime/debug"
	"sync"

	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"go.uber.org/zap"
)

// 临时组
type TGroup struct {
	State     int8                 // 组状态 1:正常，2：删除
	UsersLock sync.RWMutex         // 读写锁
	Users     map[string]bool      // 用户
	Gid       string               // 组id
	Send      chan *types.GroupMsg // 待发送的数据

}

// 初始化
func NewTGroup(gid string) (tGroup *TGroup) {
	tGroup = &TGroup{
		State: 1,
		Gid:   gid,
		Users: make(map[string]bool),
		Send:  make(chan *types.GroupMsg, Config.GroupMsgChanMax),
	}
	go tGroup.ListenSendChan()
	return
}

// 用户进入临时组
func (tGroup *TGroup) UserJoin(uid string) (ok bool) {
	tGroup.UsersLock.Lock()
	defer tGroup.UsersLock.Unlock()

	if tGroup.State == 2 {
		return false
	}

	tGroup.Users[uid] = true

	return true
}

// 用户离开临时组
func (tGroup *TGroup) UserLeave(uid string) {
	tGroup.UsersLock.Lock()
	defer tGroup.UsersLock.Unlock()
	delete(tGroup.Users, uid)

	// 如果临时组内没有用户  将临时组状态改为删除
	if len(tGroup.Users) == 0 {
		tGroup.DeleteTGroup()
	}
}

// 删除临时组
func (tGroup *TGroup) DeleteTGroup() {
	TGroupM.DeleteTGroup(tGroup)

	tGroup.CloseSend()
}

// 将群消息发送到 send chan
func (tGroup *TGroup) SendChan(groupMsg *types.GroupMsg) {
	//如果临时组被删除 则返回
	if tGroup.State == 2 {
		return
	}
	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("tGroup.SendChan error", zap.String("gid", tGroup.Gid), zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		}
	}()
	tGroup.Send <- groupMsg
}

// 关闭send chan
func (tGroup *TGroup) CloseSend() {
	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("tGroup.CloseSend error", zap.String("gid", tGroup.Gid), zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		}
	}()

	if tGroup == nil || tGroup.State == 2 {
		return
	}
	tGroup.State = 2
	close(tGroup.Send)
}

// 向临时组内用户发送消息
func (tGroup *TGroup) SendToUsers(groupMsg *types.GroupMsg) {

	//拦截恐慌
	defer func() {
		if err := recover(); err != nil {
			logger.L.Error("tGroup.SendToUsers error", zap.String("gid", tGroup.Gid), zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
		}
	}()

	tGroup.UsersLock.RLock()
	defer tGroup.UsersLock.RUnlock()
	//判断分组状态
	if tGroup.State == 2 {
		return
	}

	for uid := range tGroup.Users {
		if uid == groupMsg.ExceptUid {
			continue
		}
		client, ok := ClientM.GetClient(uid)
		if ok {
			client.SafeSendMsg(groupMsg.Msg, 2)
		}

	}

}

// 监听send chan 收到消息后向临时组内用户发送消息
func (tGroup *TGroup) ListenSendChan() {
	defer func() {
		if r := recover(); r != nil {
			logger.L.Error("tGroup.ListenSendChan stop error ", zap.String("gid", tGroup.Gid),
				zap.Any("error", r))
		}
	}()

	defer func() {
		tGroup.DeleteTGroup()
	}()

	for GroupMsg := range tGroup.Send {
		tGroup.SendToUsers(GroupMsg)
	}
}
