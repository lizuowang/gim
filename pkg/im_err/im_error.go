package im_err

type ImError struct {
	Code int
	Msg  string
}

func (e *ImError) Error() string {
	return e.Msg
}

// 错误码
const (
	OK                   = 200   // 成功
	ErrUserOffline       = 10001 // 用户不在线
	ErrUserOnline        = 10002 // 重复登录
	ErrRpcClientNotFound = 20001 // rpc客户端不存在
	ErrParams            = 20002 // 参数错误
	ErrJson              = 20003 // json解析错误
	ErrNoHandleMsg       = 20004 // 未设置处理消息函数
	ErrSendChanFull      = 20005 // 发送通道已满
)

// 错误码对应信息
var CodeMsg = map[int]string{
	ErrUserOffline:       "USER_OFFLINE",
	ErrUserOnline:        "USER_ONLINE",
	ErrRpcClientNotFound: "RPC_CLIENT_NOT_FOUND",
	ErrParams:            "PARAMS_ERROR",
	ErrJson:              "JSON_ERROR",
	ErrNoHandleMsg:       "NO_HANDLE_MSG",
	ErrSendChanFull:      "SEND_CHAN_FULL",
}

// 创建错误
func NewImError(code int) *ImError {
	return &ImError{
		Code: code,
		Msg:  CodeMsg[code],
	}
}
