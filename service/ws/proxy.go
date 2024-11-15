package ws

import (
	"github.com/lizuowang/gim/pkg/types"
)

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

var (
	Proxy RunProxy
)

func InitProxy() {

	if Config.RunMode == RunModeLocal {
		Proxy = NewLocalProxy()
	} else {
		Proxy = NewRedisProxy(Config.RedisClient)
	}

}

func GetSubMsgList() *types.WsMsgList {
	msgList := &types.WsMsgList{
		MsgNum:     0,
		ConsumeNum: 0,
	}
	if redisProxy == nil {
		return msgList
	}

	msgList.MsgNum = int64(len(redisProxy.MsgCh))
	redisProxy.Lock.RLock()
	msgList.ConsumeNum = len(redisProxy.ConsumerGm)
	redisProxy.Lock.RUnlock()

	return msgList
}
