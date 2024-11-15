package etcd_client

import (
	"fmt"
	"log"

	clientv3 "go.etcd.io/etcd/client/v3"
)

var (
	client *clientv3.Client
)

func CreateClient(conf *clientv3.Config) (err error) {
	client, err = clientv3.New(*conf)
	if err != nil {
		panic(fmt.Errorf("初始化etcd失败: %s %v", err, conf))
	}
	log.Println("初始化etcd成功", err)
	return
}

func GetClient() *clientv3.Client {
	return client
}

func Close() {
	if client == nil {
		return
	}
	log.Println("关闭etcd")
	client.Close()
}
