package ws

import (
	"time"

	"github.com/lizuowang/gim/pkg/types"
	rpcClient "github.com/lizuowang/gim/service/api/client"
)

const (
	SUB_TYPE_GROUP = 1                     //消息类型 组消息
	SUB_GROUP_KEY  = "_gim:sub:trgoup:key" //订阅key
)

type RedisProxy struct {
	LocalProxy
}

// 实例化
func NewRedisProxy() *RedisProxy {
	return &RedisProxy{}
}

/** 组消息 */

// 向组内发送消息
func (rp *RedisProxy) SendResDataToGroup(gid string, resData *types.ResponseData, exceptUid string) (err error) {

	subMsg := &types.SubMsg{
		Tgid:      gid,
		Mol:       "",
		Ctrl:      "",
		Data:      resData.A,
		ExceptUid: exceptUid,
		CTime:     time.Now().UnixMilli(), // int64 毫秒
		MsgType:   types.MSG_TYPE_GROUP,
	}

	err = ProxyMsg.PublishGlobalMsg(subMsg)
	return
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

/** 临时组 */
// 通过客户端加入临时组
func (rp *RedisProxy) JoinGroup(uid string, tgid string) (ok bool) {
	if ClientM.HasUser(uid) {
		return rp.JoinLockGroup(uid, tgid)
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}
	ok, _ = rpcClient.JoinGroup(server, uid, tgid)

	return
}

// 通过客户端退出临时组
func (rp *RedisProxy) OutGroup(uid string, tgid string) (ok bool) {
	if ClientM.HasUser(uid) {
		return rp.OutLockGroup(uid, tgid)
	}

	server, err := GetUserRpcServer(uid)
	if err != nil {
		return
	}
	ok, _ = rpcClient.OutGroup(server, uid, tgid)

	return
}
