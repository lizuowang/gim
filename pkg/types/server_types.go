package types

import "fmt"

type Server struct {
	Ip   string `json:"ip"`
	Port string `json:"port"`
}

func NewServer(ip string, port string) *Server {

	return &Server{Ip: ip, Port: port}
}

func (s *Server) String() (str string) {
	if s == nil {
		return
	}

	str = fmt.Sprintf("%s:%s", s.Ip, s.Port)

	return
}

// ws管理器信息
type WsManagerInfo struct {
	ClientsLen int `json:"clientsLen"` //客户端数量
	TGroupLen  int `json:"tGroupLen"`  //tgroup 数量
}

type WsMsgList struct {
	MsgNum     int64 `json:"msgNum"`     //堆积消息数量
	ConsumeNum int   `json:"consumeNum"` //消费者数量
	FreeCNum   int32 `json:"freeCNum"`   //空闲写成书数量
}

// 系统信息
type SysInfo struct {
	NumGoroutine int            `json:"numGoroutine"` //goroutine数量
	CPUPercent   float64        `json:"cpuPercent"`   //cpu 使用率
	MemPercent   float64        `json:"memPercent"`   //内存使用率
	NumCPU       int            `json:"numCPU"`       //cpu 数量
	ManagerInfo  *WsManagerInfo `json:"managerInfo"`
	MsgList      *WsMsgList     `json:"msgList"`    //队列消息
	SubMsgList   *WsMsgList     `json:"subMsgList"` //订阅消息
	Name         string         `json:"name"`
	Version      string         `json:"version"`
}

func NewSysInfo() *SysInfo {
	return &SysInfo{
		ManagerInfo: &WsManagerInfo{},
		MsgList:     &WsMsgList{},
	}
}

type SycCollectInfo struct {
	SysInfo
	NodeNum  int        `json:"nodeNum"`
	NodeList []*SysInfo `json:"nodeMap"`
}

func NewSycCollectInfo() *SycCollectInfo {
	return &SycCollectInfo{
		SysInfo:  *NewSysInfo(),
		NodeNum:  0,
		NodeList: make([]*SysInfo, 0),
	}
}
