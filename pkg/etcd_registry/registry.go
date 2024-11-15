package etcd_registry

import (
	"context"
	"log"

	"github.com/lizuowang/gim/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceRegister 创建租约注册服务
type ServiceRegister struct {
	cli     *clientv3.Client //etcd client
	leaseID clientv3.LeaseID //租约ID
	//租约keepalieve相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	name          string //服务名
	Addr          string //服务地址
}

// NewServiceRegister 新建注册服务
func NewServiceRegister(cli *clientv3.Client, name, addr string, lease int64) (*ServiceRegister, error) {

	ser := &ServiceRegister{
		cli:  cli,
		name: name,
		Addr: addr,
	}

	//申请租约设置时间keepalive
	if err := ser.putKeyWithLease(lease); err != nil {
		return nil, err
	}

	return ser, nil
}

// 设置租约
func (s *ServiceRegister) putKeyWithLease(lease int64) error {
	//设置租约时间
	resp, err := s.cli.Grant(context.Background(), lease)
	if err != nil {
		return err
	}
	//注册服务并绑定租约
	_, err = s.cli.Put(context.Background(), s.name, s.Addr, clientv3.WithLease(resp.ID))
	if err != nil {
		return err
	}
	//设置续租 定期发送需求请求
	leaseRespChan, err := s.cli.KeepAlive(context.Background(), resp.ID)

	if err != nil {
		return err
	}
	s.leaseID = resp.ID
	s.keepAliveChan = leaseRespChan
	logger.L.Info("注册服务", zap.String("service_name", s.name), zap.String("addr", s.Addr))
	return nil
}

// ListenLeaseRespChan 监听 续租情况
func (s *ServiceRegister) ListenLeaseRespChan(f func(leaseKeepResp *clientv3.LeaseKeepAliveResponse)) {
	for leaseKeepResp := range s.keepAliveChan {
		f(leaseKeepResp)
	}
	log.Println("关闭服务续租", s.name)
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	//撤销租约
	if _, err := s.cli.Revoke(context.Background(), s.leaseID); err != nil {
		return err
	}
	logger.L.Info("注销服务", zap.String("service_name", s.name), zap.String("addr", s.Addr))
	return s.cli.Close()
}
