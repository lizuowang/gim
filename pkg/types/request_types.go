package types

/************************  请求数据  **************************/
// 通用请求数据格式
type Request struct {
	Seq  string                 `json:"seq"`
	Cmd  string                 `json:"cmd"`
	Data map[string]interface{} `json:"data,omitempty"`
}

type RequestProxy struct {
	Mod   string            `json:"mod"`
	Ctrl  string            `json:"ctrl"`
	Param map[string]string `json:"param,omitempty"`
}
