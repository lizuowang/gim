package etcd_registry

import (
	"context"
	"log"
	"time"

	"github.com/lizuowang/gim/pkg/logger"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceRegister 创建租约注册服务
type ServiceRegister struct {
	cli     *clientv3.Client //etcd client
	lease   int64            //租约时间
	status  int              //状态 1 注册中 2 注册成功 3 关闭
	leaseID clientv3.LeaseID //租约ID
	//租约keepalieve相应chan
	keepAliveChan <-chan *clientv3.LeaseKeepAliveResponse
	name          string //服务名
	Addr          string //服务地址
}

// NewServiceRegister 新建注册服务
func NewServiceRegister(cli *clientv3.Client, name, addr string, lease int64) (*ServiceRegister, error) {

	ser := &ServiceRegister{
		cli:    cli,
		lease:  lease,
		status: 1,
		name:   name,
		Addr:   addr,
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
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	resp, err := s.cli.Grant(ctx, lease)
	if err != nil {
		return err
	}
	//注册服务并绑定租约
	_, err = s.cli.Put(ctx, s.name, s.Addr, clientv3.WithLease(resp.ID))
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
	s.status = 2
	logger.L.Info("注册服务", zap.String("service_name", s.name), zap.String("addr", s.Addr))
	return nil
}

// ListenLeaseRespChan 监听 续租情况
func (s *ServiceRegister) ListenLeaseRespChan(f func(leaseKeepResp *clientv3.LeaseKeepAliveResponse)) {
	for leaseKeepResp := range s.keepAliveChan {
		f(leaseKeepResp)
	}
	logger.L.Info("关闭服务续租", zap.String("service_name", s.name))
	// 如果是重试续租
	for {
		// 如果状态为关闭，则退出
		if s.status == 3 {
			break
		}
		err := s.putKeyWithLease(s.lease)
		if err != nil {
			logger.L.Error("etcd_registry 重试续租失败", zap.String("service_name", s.name), zap.Error(err))
			time.Sleep(time.Second * 2)
			continue
		}
		logger.L.Info("etcd_registry 重试续租成功", zap.String("service_name", s.name))
		go s.ListenLeaseRespChan(f)
		break
	}

	log.Println("关闭服务续租", s.name)
}

// Close 注销服务
func (s *ServiceRegister) Close() error {
	// 设置状态为关闭
	s.status = 3
	//撤销租约
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := s.cli.Revoke(ctx, s.leaseID); err != nil {
		log.Println("etcd_registry 撤销租约失败", zap.String("service_name", s.name), zap.Error(err))
		return err
	}
	logger.L.Info("注销服务", zap.String("service_name", s.name), zap.String("addr", s.Addr))
	return s.cli.Close()
}
