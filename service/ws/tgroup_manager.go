package ws

import (
	"sync"

	"github.com/lizuowang/gim/pkg/types"
)

// 临时组管理
type TGroupManager struct {
	TGroupsLock sync.RWMutex       // 读写锁
	TGroups     map[string]*TGroup // 全部组

}

func NewTGroupManager() (tGroupManager *TGroupManager) {
	tGroupManager = &TGroupManager{
		TGroups: make(map[string]*TGroup),
	}
	return
}

// 获取或者创建一个组
func (tgm *TGroupManager) GetOrCreateTGroup(gid string) (tGroup *TGroup) {
	// 获取一个已有的组
	tGroup, ok := tgm.GetTGroup(gid)
	if ok {
		return
	}

	tgm.TGroupsLock.Lock()
	defer tgm.TGroupsLock.Unlock()

	tGroup, ok = tgm.TGroups[gid]
	if ok {
		return
	}

	// 创建一个组
	tGroup = NewTGroup(gid)
	tgm.TGroups[gid] = tGroup

	return
}

// 获取一个临时组
func (tgm *TGroupManager) GetTGroup(gid string) (tGroup *TGroup, ok bool) {

	tgm.TGroupsLock.RLock()
	defer tgm.TGroupsLock.RUnlock()

	tGroup, ok = tgm.TGroups[gid]
	return
}

// 删除一个临时组
func (tgm *TGroupManager) DeleteTGroup(tGroup *TGroup) {
	tgm.TGroupsLock.Lock()
	defer tgm.TGroupsLock.Unlock()

	oTGroup, ok := tgm.TGroups[tGroup.Gid]
	if !ok {
		return
	}
	if oTGroup != tGroup {
		return
	}

	delete(tgm.TGroups, tGroup.Gid)
}

// 用户进入一个临时组
func (tgm *TGroupManager) UserJoinTGroup(gid string, uid string) (ok bool) {
	tGroup := tgm.GetOrCreateTGroup(gid)
	ok = tGroup.UserJoin(uid)
	if !ok {
		ok = tgm.UserJoinTGroup(gid, uid)
	}
	return
}

// 用户离开一个临时组
func (tgm *TGroupManager) UserOutTGroup(gid string, uid string) bool {
	tGroup, ok := tgm.GetTGroup(gid)
	if !ok {
		return false
	}
	tGroup.UserLeave(uid)

	return true
}

// 向一个临时组内发送消息
func (tgm *TGroupManager) SendMessageToTGroup(gid string, groupMsg *types.GroupMsg) (err error) {
	tGroup, ok := tgm.GetTGroup(gid)
	if !ok {
		return
	}
	tGroup.SendChan(groupMsg)

	return
}
