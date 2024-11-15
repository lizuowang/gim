package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime/debug"

	"github.com/lizuowang/gim/pkg/im_err"
	"github.com/lizuowang/gim/pkg/logger"
	"github.com/lizuowang/gim/pkg/types"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	userOnlinePrefix    = "_mws:user:online:" // 用户在线状态
	userOnlineCacheTime = 10
)

/*********************  查询用户是否在线  ************************/
func getUserOnlineKey(userKey string) (key string) {

	key = fmt.Sprintf("%s%s%s", Config.RedisPrefix, userOnlinePrefix, userKey)

	return
}

// 设置并获取原用户在线数据
func GetSetUserOnlineInfo(userKey string, userOnline *UserOnline) (oldUserOnline *UserOnline, err error) {
	key := getUserOnlineKey(userKey)

	valueByte, err := json.Marshal(userOnline)
	if err != nil {
		return
	}

	oldData, err := Config.RedisClient.GetSet(context.Background(), key, valueByte).Result()
	if err != nil && err != redis.Nil {
		return
	}
	err = nil
	//设置过期时间
	expTime := GetUserOnlineExpTime()
	Config.RedisClient.Do(context.Background(), "expire", key, expTime).Result()

	if oldData != "" {
		oldUserOnline = &UserOnline{}
		err = json.Unmarshal([]byte(oldData), oldUserOnline)
		if err != nil {
			return
		}
	}

	return
}

// 获取用户缓存时间
func GetUserOnlineExpTime() (expTIme int) {
	expTIme = 3600

	if Config.KeepTime > 0 {
		expTIme = Config.KeepTime + userOnlineCacheTime
	}

	return
}

// 设置用户缓存时间
func ResetUserOnlineExpTime(userKey string) (err error) {
	key := getUserOnlineKey(userKey)

	expTIme := GetUserOnlineExpTime()
	_, err = Config.RedisClient.Do(context.Background(), "expire", key, expTIme).Result()

	return
}

// 删除用户在线数据
func DelUserOnlineInfo(userKey string) (err error) {
	key := getUserOnlineKey(userKey)

	_, err = Config.RedisClient.Do(context.Background(), "del", key).Result()

	return
}

// 获取用户在线信息
func GetUserOnlineInfo(userKey string) (userOnline *UserOnline, err error) {

	key := getUserOnlineKey(userKey)

	data, err := Config.RedisClient.Get(context.Background(), key).Bytes()
	if err != nil {
		if err != redis.Nil {
			logger.L.Error("GetUserOnlineInfo get error ", zap.Any("error", err), zap.String("debug_stack", string(debug.Stack())))
			return
		}

		return
	}

	userOnline = &UserOnline{}
	err = json.Unmarshal(data, userOnline)
	if err != nil {
		return
	}

	return
}

// 获取用户服务信息
func GetUserRpcServer(uid string) (server *types.Server, err error) {
	userInfo, err := GetUserOnlineInfo(uid)
	if err != nil {
		return
	}

	if userInfo.AccIp == "" || userInfo.RpcPort == "" {
		err = im_err.NewImError(im_err.ErrUserOffline)
		return
	}

	server = types.NewServer(userInfo.AccIp, userInfo.RpcPort)
	return
}
