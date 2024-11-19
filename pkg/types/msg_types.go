package types

// 单发消息
type SendMsg struct {
	Sync     int8      `json:"sync"`     // 是否同步 1:同步 2:异步
	Response *Response `json:"response"` // 消息
}

// 获取一个单发消息
func NewSendMsg(response *Response, sync int8) (sendMsg *SendMsg) {

	sendMsg = &SendMsg{
		Sync:     sync,
		Response: response,
	}

	return
}

// 群发消息
type GroupMsg struct {
	ExceptUid string `json:"exceptUid"` //排除一个uid
	Msg       []byte `json:"msg"`       // 消息
}

// 获取一个群发消息
func NewGroupMsg(msg []byte, exceptUid string) (groupMsg *GroupMsg) {

	groupMsg = &GroupMsg{
		ExceptUid: exceptUid,
		Msg:       msg,
	}

	return
}

// 获取一个组消息
func GetResDataGroupMsg(resData *ResponseData, exceptUid string) (groupMsg *GroupMsg, err error) {
	response := NewResponse("", "", MSG_TYPE_GROUP, resData)
	msgBytes, err := response.Bytes()
	if err != nil {
		return
	}
	groupMsg = NewGroupMsg(msgBytes, exceptUid)
	return
}

// 订阅消息
type SubMsg struct {
	Tgid      string `json:"tgid"`       //tgid
	Mol       string `json:"mol"`        //mod
	Ctrl      string `json:"ctrl"`       //ctrl
	Data      *AData `json:"data"`       // 数据
	ExceptUid string `json:"except_uid"` //排除一个uid
	CTime     int64  `json:"c_time"`     //毫秒时间戳
	MsgType   int8   `json:"msg_type"`   //消息类型 1:消息
	ToUid     string `json:"to_uid"`     //到组内用户
}
