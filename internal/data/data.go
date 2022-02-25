package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	redis "github.com/go-redis/redis/v8"
	"github.com/google/wire"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewRedisRepo)

// Data .
type Data struct {
	// TODO wrapped database client
	// redis连接客户端
	*redis.ClusterClient
}

// NewData 实例化redis数据库连接对象
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	data := new(Data)
	data.ClusterClient = redis.NewFailoverClusterClient(&redis.FailoverOptions{
		MasterName:            c.Redis.MasterName,
		SentinelAddrs:         []string{fmt.Sprintf("%s:%d", c.Redis.Host, c.Redis.SentinelPort)},
		RouteByLatency:        false,
		RouteRandomly:         false,
		SlaveOnly:             false,
		UseDisconnectedSlaves: false,
		DB:                    0,
		PoolSize:              int(c.Redis.PoolSize),
		MinIdleConns:          int(c.Redis.MinIdleConns),
	})
	helper := log.NewHelper(logger)

	if err := data.Ping(context.Background()).Err(); err != nil {
		helper.Errorf("redis数据库连接失败,失败信息:%s\n", err)
		return nil, nil, err
	}

	cleanup := func() {
		err := data.Close()
		if err != nil {
			helper.Errorf("redis数据库连接关闭失败,失败信息:%s\n", err)
			return
		}
		helper.Info("redis数据库连接关闭成功\n")
	}

	return data, cleanup, nil
}
