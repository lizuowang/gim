package types

import (
	"encoding/json"
)

const (
	MSG_TYPE_RESPONSE = 1 //相应消息
	MSG_TYPE_GROUP    = 2 //群消息
	MSG_TYPE_U2U      = 3 //用户对用户
)

/************************  响应数据  **************************/
type Response struct {
	Seq     string        `json:"seq"`      // 消息的Id
	Cmd     string        `json:"cmd"`      // 消息的cmd 动作
	MsgType int           `json:"msgType"`  // 消息的
	RData   *ResponseData `json:"response"` // 消息体
}

type AData map[string]interface{}

type ResponseData struct {
	S int    `json:"s"`
	A *AData `json:"a"`
}

// 系统信息
type System struct {
	// Sys    *SysTime  `json:"sys"`    // 系统时间
	Errror *SysError `json:"errror"` // 系统错误
}

type SysTime struct {
	Time int64 `json:"time"` //系统时间
}

type SysError struct {
	Type int    `json:"type"`
	Msg  string `json:"msg"`
}

// 获取一个响应
func NewResponse(seq string, cmd string, msgType int, rData *ResponseData) *Response {

	return &Response{Seq: seq, Cmd: cmd, MsgType: msgType, RData: rData}
}

func (r *Response) String() (headStr string) {
	headBytes, _ := json.Marshal(r)
	headStr = string(headBytes)

	return
}

func (r *Response) Bytes() (headBytes []byte, err error) {
	headBytes, err = json.Marshal(r)
	return
}

// 响应数据数据
func NewResponseData(a *AData, params ...int) *ResponseData {

	s := 1
	if len(params) > 0 {
		s = params[0]
	}
	return &ResponseData{S: s, A: a}
}

// 组装响应数据
func NewCtrlResponseData(ad interface{}, mol, ctrl string) *ResponseData {
	aData := HandleAData(ad, mol, ctrl)
	return NewResponseData(aData)
}

// 组装响应数据
func NewMolResponseData(ad interface{}, mol string) *ResponseData {
	aData := HandleMolAData(ad, mol)
	return NewResponseData(aData)
}

// 获取系统信息
func NewSystem(msg string, ty ...int) (system *System) {
	system = &System{
		// Sys: &SysTime{Time: time.Now().Unix()},
	}
	if msg != "" {
		err_type := 0
		if len(ty) > 0 {
			err_type = ty[0]
		}
		system.Errror = &SysError{Type: err_type, Msg: msg}

	}

	return
}

func NewErrAData(msg string, ty int) (errAData *AData) {
	errAData = &AData{"system": NewSystem(msg, ty)}
	return
}

// 组装数据
func HandleAData(ad any, mol, ctrl string) (aData *AData) {

	aData = &AData{mol: &AData{ctrl: ad}}
	return
}

// 组装数据
func HandleMolAData(ad interface{}, mol string) (aData *AData) {
	aData = &AData{mol: ad}
	return
}
