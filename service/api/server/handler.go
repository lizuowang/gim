package server

import (
	"context"
	"encoding/json"

	"github.com/lizuowang/gim/pkg/im_err"
	rpc "github.com/lizuowang/gim/service/api/server/kitex_gen/im/rpc"
	"github.com/lizuowang/gim/service/sys"
	"github.com/lizuowang/gim/service/ws"
)

// ImRpcImpl implements the last service interface defined in the IDL.
type ImRpcImpl struct{}

// Hello implements the ImRpcImpl interface.
func (s *ImRpcImpl) Hello(ctx context.Context, req *rpc.Request) (resp *rpc.Response, err error) {
	// TODO: Your code here...

	resp = &rpc.Response{
		Message: "success:" + "_aaaa", // Use the correct field name here
	}
	return
}

// 发送消息到用户
func (s *ImRpcImpl) SendMsgToUser(ctx context.Context, req *rpc.SendMsgReq) (resp *rpc.SendMsgRsp, err error) {
	uid := req.GetUid()

	err = ws.Proxy.SendMsgToLocalUser(uid, req.GetMsg())
	resp = &rpc.SendMsgRsp{}
	if err != nil {
		resp.RetCode = 0
	} else {
		resp.RetCode = im_err.OK
	}

	return resp, err
}

// 发送消息到组内用户
func (s *ImRpcImpl) SendMsgToUserByTgid(ctx context.Context, req *rpc.SendMsgByTgidReq) (resp *rpc.SendMsgRsp, err error) {
	uid := req.GetUid()

	err = ws.Proxy.SendMsgToLocalUserByTgid(uid, req.GetTgid(), req.GetMsg())
	resp = &rpc.SendMsgRsp{}
	if err != nil {
		resp.RetCode = 0
	} else {
		resp.RetCode = im_err.OK
	}

	return resp, err
}

// 停止一个用户
func (s *ImRpcImpl) StopUserClient(ctx context.Context, req *rpc.UserIdReq) (resp *rpc.StopUserClientRsp, err error) {
	ws.Proxy.StopLocalUserClient(req.Uid)
	resp = &rpc.StopUserClientRsp{Res: true}
	return resp, err
}

// 获取服务系统信息
func (s *ImRpcImpl) GetSysInfo(ctx context.Context) (resp *rpc.BytesRsp, err error) {
	//获取信息
	sysInfo := sys.GetSysInfo()
	//序列化
	bytes, err := json.Marshal(sysInfo)
	if err != nil {
		return nil, err
	}
	resp = &rpc.BytesRsp{
		Data: bytes,
	}
	return resp, err
}
