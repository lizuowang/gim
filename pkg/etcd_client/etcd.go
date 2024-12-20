package etcd_client

import (
	"context"
	"fmt"
	"log"
	"time"

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

	//检查etcd是否健康
	// Try a simple operation to confirm the connection
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	for _, ep := range conf.Endpoints {
		_, err := client.Status(ctx, ep)
		if err != nil {
			log.Printf("Failed to get status from %s: %v", ep, err)
			panic(fmt.Errorf("初始化etcd失败: %s %v", ep, err))
		}
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
