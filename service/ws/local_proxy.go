package ws

import "github.com/lizuowang/gim/pkg/types"

type LocalProxy struct {
}

// 实例化
func NewLocalProxy() *LocalProxy {
	lp := &LocalProxy{}
	return lp
}

// 向临时组内发送消
func (lp *LocalProxy) SendResDataToGroup(gid string, resData *types.ResponseData, exceptUid string) (err error) {
	err = lp.SendResDataToLocalGroup(gid, resData, exceptUid)
	return
}

// 向本地临时组内发送消
func (lp *LocalProxy) SendResDataToLocalGroup(gid string, resData *types.ResponseData, exceptUid string) (err error) {
	groupMsg, err := types.GetResDataGroupMsg(resData, exceptUid)
	if err != nil {
		return
	}
	err = TGroupM.SendMessageToTGroup(gid, groupMsg)
	return
}

// 向组内发送消息
func (lp *LocalProxy) SendMsgToLocalGroup(gid string, groupMsg *types.GroupMsg) (err error) {
	err = TGroupM.SendMessageToTGroup(gid, groupMsg)
	return
}

// 向用户发消息
func (lp *LocalProxy) SendResDataToUser(uid string, resData *types.ResponseData) (err error) {
	err = lp.SendResDataToLocalUser(uid, resData)
	return
}

// 向本地用户发送消息
func (lp *LocalProxy) SendResDataToLocalUser(uid string, resData *types.ResponseData) (err error) {
	response := types.NewResponse("", "", types.MSG_TYPE_U2U, resData)
	err = ClientM.SendResponseToUser(uid, response)
	return
}

// 向组内用户发送消息
func (lp *LocalProxy) SendResDataToUserByTgid(uid string, tgid string, resData *types.ResponseData) (err error) {
	err = lp.SendResDataToLocalUserByTgid(uid, tgid, resData)
	return
}

// 向组内用户发送消息
func (lp *LocalProxy) SendResDataToLocalUserByTgid(uid string, tgid string, resData *types.ResponseData) (err error) {
	response := types.NewResponse("", "", types.MSG_TYPE_U2U, resData)
	err = ClientM.SendResponseToUserByTgid(uid, tgid, response)
	return
}

// 向用户发送消息
func (lp *LocalProxy) SendMsgToUser(uid string, msg []byte) (err error) {
	err = lp.SendMsgToLocalUser(uid, msg)
	return
}

// 向本地用户发送消息
func (lp *LocalProxy) SendMsgToLocalUser(uid string, msg []byte) (err error) {
	err = ClientM.SendMsgToUser(uid, msg)
	return
}

// 向组内用户发送消息
func (lp *LocalProxy) SendMsgToUserByTgid(uid string, tgid string, msg []byte) (err error) {
	err = lp.SendMsgToLocalUserByTgid(uid, tgid, msg)
	return
}

// 向本地组内用户发送消息
func (lp *LocalProxy) SendMsgToLocalUserByTgid(uid string, tgid string, msg []byte) (err error) {
	err = ClientM.SendMsgToUserByTgid(uid, tgid, msg)
	return
}

// 停止用户client
func (lp *LocalProxy) StopLocalUserClient(uid string) {
	ClientM.StopClient(uid)
}

// 停止用户client
func (lp *LocalProxy) StopUserClient(userOnline *UserOnline) {
	lp.StopLocalUserClient(userOnline.UserId)
}
