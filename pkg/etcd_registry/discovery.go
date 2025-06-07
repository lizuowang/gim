package etcd_registry

import (
	"context"
	"log"

	"github.com/lizuowang/gim/pkg/logger"
	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

// ServiceDiscovery 服务发现
type ServiceDiscovery struct {
	cli          *clientv3.Client          //etcd client
	name         string                    //服务名
	onRegister   func(kv *mvccpb.KeyValue) //注册回调
	onUnregister func(kv *mvccpb.KeyValue) //注销回调
}

// NewServiceDiscovery  新建发现服务
func NewServiceDiscovery(cli *clientv3.Client, name string, onRegister func(kv *mvccpb.KeyValue), onUnregister func(kv *mvccpb.KeyValue)) *ServiceDiscovery {
	return &ServiceDiscovery{
		cli:          cli,
		name:         name,
		onRegister:   onRegister,
		onUnregister: onUnregister,
	}
}

// WatchService 初始化服务列表和监视
func (s *ServiceDiscovery) WatchService() error {

	//根据前缀获取现有的key，同时获取当前revision
	resp, err := s.cli.Get(context.Background(), s.name, clientv3.WithPrefix())
	if err != nil {
		return err
	}

	for _, ev := range resp.Kvs {
		s.onRegister(ev)
	}

	//从获取服务时的revision+1开始监视，确保不漏掉任何变更
	go s.watcherFromRevision(resp.Header.Revision + 1)

	return nil
}

// watcher 监听前缀
func (s *ServiceDiscovery) watcher() {
	rch := s.cli.Watch(context.Background(), s.name, clientv3.WithPrefix())
	logger.L.Info("ServiceDiscovery.watcher", zap.String("prefix", s.name))
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.onRegister(ev.Kv)
			case mvccpb.DELETE: //删除
				s.onUnregister(ev.Kv)
			}
		}
	}
	log.Println("关闭服务发现", s.name)
}

// watcherFromRevision 监听前缀，从指定revision开始
func (s *ServiceDiscovery) watcherFromRevision(revision int64) {
	rch := s.cli.Watch(context.Background(), s.name, clientv3.WithPrefix(), clientv3.WithRev(revision))
	logger.L.Info("ServiceDiscovery.watcherFromRevision", zap.String("prefix", s.name), zap.Int64("revision", revision))
	for wresp := range rch {
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT: //修改或者新增
				s.onRegister(ev.Kv)
			case mvccpb.DELETE: //删除
				s.onUnregister(ev.Kv)
			}
		}
	}
	log.Println("关闭服务发现", s.name)
}
