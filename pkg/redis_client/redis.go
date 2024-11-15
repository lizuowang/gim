package redis_client

import (
	"context"
	"fmt"
	"log"

	"github.com/redis/go-redis/v9"
)

var (
	client *redis.Client
)

func CreateClient(conf *redis.Options) {

	client = redis.NewClient(conf)

	pong, err := client.Ping(context.Background()).Result()
	if err != nil {
		panic(fmt.Errorf("redis PONG Fatal error: %s", err))
	}
	log.Println("初始化redis:", pong, err)
	// Output: PONG <nil>
}

func GetClient() (c *redis.Client) {

	return client
}

func Close() {
	if client == nil {
		return
	}
	log.Println("关闭redis")
	client.Close()
}
