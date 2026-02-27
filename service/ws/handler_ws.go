package ws

import (
	"encoding/json"
	"net/http"
	"runtime/debug"

	"github.com/lizuowang/gim/pkg/helper"
	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lxzan/gws"
	"go.uber.org/zap"
)

type Handler struct {
	gws.BuiltinEventHandler
	HandleRequestMsg func(client *Client, request *types.Request) (aData *types.AData, err error) //处理消息
}

func (c *Handler) OnPing(socket *gws.Conn, payload []byte) {
	client, ok := ClientM.GetClient(c.GetUid(socket))
	if ok {
		client.Heartbeat() //心跳
	} else {
		socket.NetConn().Close() //关闭连接
	}
	// _ = socket.WritePong(nil) //发送pong
}

func (c *Handler) OnPong(socket *gws.Conn, payload []byte) {

}

func (c *Handler) OnMessage(socket *gws.Conn, gwsMsg *gws.Message) {
	defer func() {
		if r := recover(); r != nil {
			logger.L.Error("ws.OnMessage  error ", zap.String("uid", c.GetUid(socket)), zap.Any("error", r), zap.String("debug_stack", string(debug.Stack())))
		}
	}()
	//获取client
	client, ok := ClientM.GetClientByConn(socket)
	if !ok {
		logger.L.Error("ws.OnMessage 获取client失败 ", zap.String("uid", c.GetUid(socket)), zap.String("addr", socket.RemoteAddr().String()))
		socket.NetConn().Close()
		return
	}

	// 未设置处理消息函数
	if c.HandleRequestMsg == nil {
		logger.L.Error("ws.OnMessage 未设置处理消息函数 ", zap.String("uid", client.GetKey()))
		im_err := im_err.NewImError(im_err.ErrNoHandleMsg)
		client.SendErr(im_err.Msg, 0, "", "")
		return
	}

	//解析数据
	request := &types.Request{}
	message := gwsMsg.Bytes()
	err := json.Unmarshal(message, request)
	if err != nil {
		logger.L.Error("ws.OnMessage 获取请求参数错误 ", zap.String("uid", client.GetKey()),
			zap.Any("error", err), zap.ByteString("message", message))
		im_err := im_err.NewImError(im_err.ErrJson)
		client.SendErr(im_err.Msg, 0, request.Seq, request.Cmd)
		return
	}

	//处理消息
	var (
		aData *types.AData
	)
	aData, err = c.HandleRequestMsg(client, request)
	if err != nil {
		client.SendErr(err.Error(), 0, request.Seq, request.Cmd)
		return
	}

	//组装响应数据
	responseData := types.NewResponseData(aData)
	client.SendResponseData(responseData, request.Seq, request.Cmd, types.MSG_TYPE_RESPONSE)
}

func (c *Handler) OnClose(socket *gws.Conn, err error) {
	client, ok := ClientM.GetClient(c.GetUid(socket))
	closeClient := false
	if ok {
		closeClient = client.OnClose()
	}

	//记录日志
	if err != nil {
		logger.L.Error("ws.OnClose 关闭连接 ", zap.String("uid", c.GetUid(socket)), zap.String("addr", socket.RemoteAddr().String()), zap.Bool("closeClient", closeClient), zap.Any("error", err))
	} else {
		logger.L.Info("ws.OnClose 关闭连接 ", zap.String("uid", c.GetUid(socket)), zap.String("addr", socket.RemoteAddr().String()), zap.Bool("closeClient", closeClient))
	}

}

func (c *Handler) GetUid(socket *gws.Conn) string {
	return GetUid(socket)
}

// 获取uid
func GetUid(socket *gws.Conn) string {
	uid, ok := socket.Session().Load("uid")
	if !ok {
		return ""
	}
	uidStr, ok := uid.(string)
	if !ok || uidStr == "" {
		return ""
	}
	return uidStr
}

func HandlerUp(writer http.ResponseWriter, request *http.Request) {
	socket, err := upgrader.Upgrade(writer, request)
	if err != nil {
		return
	}

	// 获取uid
	uid := GetUid(socket)
	if uid == "" {
		socket.NetConn().Close()
		logger.L.Error("ws.HandlerUp 获取uid失败 ", zap.String("addr", socket.RemoteAddr().String()))
		return
	}

	// 唯一登录
	userOnline, err := uniqueLogin(uid, socket)
	if err != nil {
		socket.NetConn().Close()
		logger.L.Error("ws.HandlerUp 唯一登录失败 ", zap.String("uid", uid), zap.String("addr", socket.RemoteAddr().String()), zap.Any("error", err))
		return
	}

	logger.L.Info("ws.HandlerUp 建立连接 ", zap.String("uid", uid), zap.String("addr", socket.RemoteAddr().String()))

	//获取get参数msg_type
	getMsgCode := request.URL.Query().Get("msg_code")
	msgCode := gws.OpcodeBinary
	if getMsgCode == "1" {
		msgCode = gws.OpcodeText
	}

	//初始化客户端
	client := NewClient(socket, uid, userOnline.RandUuid, msgCode)
	client.InitStart()

	go func() {
		socket.ReadLoop()
	}()

}

// 用户在线状态
type UserOnline struct {
	AccIp     string `json:"accIp"`     // acc Ip
	ClientArr string `json:"clientArr"` // 客户端地址
	AccPort   string `json:"accPort"`   // acc 端口
	RpcPort   string `json:"rpcPort"`   // rpc端口
	UserId    string `json:"userId"`    // 用户Id
	RandUuid  string `json:"randUuid"`  // 随机uuid
}

// 用户服务
type UserServer struct {
	Ip   string `json:"ip"`   // ip
	Port string `json:"port"` // 端口
}

func NewUserOnline(uid string, randUuid string, clientAddr string) *UserOnline {
	return &UserOnline{
		UserId:    uid,
		RandUuid:  randUuid,
		AccIp:     serverIp,
		AccPort:   Config.WsPort,
		RpcPort:   Config.RpcPort,
		ClientArr: clientAddr,
	}
}

// 用户是否在本台机器上
func (u *UserOnline) UserIsLocal() (result bool) {

	if u.AccIp == serverIp && u.AccPort == Config.WsPort {
		result = true

		return
	}

	return
}

// 唯一登录
func uniqueLogin(uid string, socket *gws.Conn) (userOnline *UserOnline, err error) {

	//是否已经在本机登录
	client, ok := ClientM.GetClient(uid)
	if ok {
		client.Close()
		err = im_err.NewImError(im_err.ErrUserOnline)
		return
	}

	// 生成唯一id
	randUuidStr := helper.CreateUniqueId()
	clientAddr := socket.RemoteAddr().String()

	//唯一登录
	userOnline = NewUserOnline(uid, randUuidStr, clientAddr)
	oldUserOnline, err := GetSetUserOnlineInfo(uid, userOnline)
	if err != nil {
		DelUserOnlineInfo(uid)
		return
	}
	if oldUserOnline != nil && oldUserOnline.RandUuid != userOnline.RandUuid {
		DelUserOnlineInfo(uid)
		err = im_err.NewImError(im_err.ErrUserOnline)
		//停止旧的连接
		Proxy.StopUserClient(oldUserOnline)
		return
	}

	return userOnline, nil
}
