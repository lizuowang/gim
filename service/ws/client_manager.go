package ws

import (
	"sync"

	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/lxzan/gws"
)

// 客户端管理器
type ClientManager struct {
	ClientsLock sync.RWMutex       // 读写锁
	Clients     map[string]*Client // 全部的连接

	ConnIdxLock   sync.RWMutex          // 连接索引锁
	ConnClientIdx map[*gws.Conn]*Client // 连接索引

}

// 初始化客户端管理器
func NewClientManager() (clientManager *ClientManager) {
	clientManager = &ClientManager{
		Clients:       make(map[string]*Client),
		ConnClientIdx: make(map[*gws.Conn]*Client),
	}

	return
}

// 连接事件
func (manager *ClientManager) OnConnect(client *Client) {
	manager.AddClient(client)
	manager.AddClientConn(client)
}

// 关闭一个客户端
func (manager *ClientManager) CloseClient(uid string) {
	client, ok := manager.GetClient(uid)
	if !ok {
		return
	}
	client.Close()
}

// 添加客户端
func (manager *ClientManager) AddClient(client *Client) {
	manager.ClientsLock.Lock()
	defer manager.ClientsLock.Unlock()

	manager.Clients[client.UserId] = client
}

// 添加客户端连接
func (manager *ClientManager) AddClientConn(client *Client) {
	manager.ConnIdxLock.Lock()
	defer manager.ConnIdxLock.Unlock()

	manager.ConnClientIdx[client.Socket] = client
}

// 删除客户端
func (manager *ClientManager) DelClient(uid string) {
	manager.ClientsLock.Lock()
	defer manager.ClientsLock.Unlock()

	delete(manager.Clients, uid)
}

// 删除客户端连接
func (manager *ClientManager) DelClientConn(client *Client) {
	manager.ConnIdxLock.Lock()
	defer manager.ConnIdxLock.Unlock()

	delete(manager.ConnClientIdx, client.Socket)
}

// 获取客户端
func (manager *ClientManager) GetClient(uid string) (client *Client, ok bool) {
	manager.ClientsLock.RLock()
	defer manager.ClientsLock.RUnlock()

	client, ok = manager.Clients[uid]
	return
}

// 获取客户端连接
func (manager *ClientManager) GetClientByConn(conn *gws.Conn) (client *Client, ok bool) {
	manager.ConnIdxLock.RLock()
	defer manager.ConnIdxLock.RUnlock()

	client, ok = manager.ConnClientIdx[conn]
	return
}

// 停止用户
func (manager *ClientManager) StopClient(uid string) {
	client, ok := manager.GetClient(uid)
	if ok {
		client.Close()
	}
}

// 向一个用户发送消息
func (manager *ClientManager) SendResponseToUser(uid string, response *types.Response) error {
	client, ok := manager.GetClient(uid)
	if !ok {
		return im_err.NewImError(im_err.ErrUserOffline)
	}

	return client.SendResponseMsg(response, 2)
}

// 向用户发送消息 如果订阅了临时组
func (manager *ClientManager) SendResponseToUserByTgid(uid string, tgid string, response *types.Response) error {
	client, ok := manager.GetClient(uid)
	if !ok {
		return im_err.NewImError(im_err.ErrUserOffline)
	}

	_, tok := client.TGroup[tgid]
	if !tok {
		return nil
	}

	return client.SendResponseMsg(response, 2)
}

// 发送消息到用户
func (manager *ClientManager) SendMsgToUser(uid string, msg []byte) error {
	client, ok := manager.GetClient(uid)
	if !ok {
		return im_err.NewImError(im_err.ErrUserOffline)
	}

	return client.SafeSendMsg(msg, 2)
}

// 发送消息到组内用户
func (manager *ClientManager) SendMsgToUserByTgid(uid string, tgid string, msg []byte) error {
	client, ok := manager.GetClient(uid)
	if !ok {
		return im_err.NewImError(im_err.ErrUserOffline)
	}
	_, tok := client.TGroup[tgid]
	if !tok {
		return nil
	}

	return client.SafeSendMsg(msg, 2)
}

// 用户是否存在
func (manager *ClientManager) HasUser(uid string) bool {
	_, ok := manager.GetClient(uid)
	return ok
}

func (manager *ClientManager) GetClientsLen() (clientsLen int) {
	manager.ClientsLock.RLock()
	defer manager.ClientsLock.RUnlock()
	clientsLen = len(manager.Clients)
	return
}

// 获取管理者信息
func GetManagerInfo() (managerInfo *types.WsManagerInfo) {
	managerInfo = &types.WsManagerInfo{
		ClientsLen: ClientM.GetClientsLen(), // 客户端连接数
		TGroupLen:  len(TGroupM.TGroups),    // 临时组数量

	}

	return
}
