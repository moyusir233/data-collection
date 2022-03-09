package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	redis "github.com/go-redis/redis/v8"
	"github.com/google/wire"
	"time"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewData, NewRedisRepo)

// Data .
type Data struct {
	// redis连接客户端
	*redis.ClusterClient
	// TS的RETENTION参数，决定了一个TS保存多长时间跨度的数据
	retention time.Duration
}

// NewData 实例化redis数据库连接对象
func NewData(c *conf.Data, logger log.Logger) (*Data, func(), error) {
	data := new(Data)

	// 实例化用于连接redis集群的客户端
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
	// 配置ts的保留时间
	data.retention = c.Redis.TsRetention.AsDuration()

	// 实例化临时日志对象
	helper := log.NewHelper(logger)

	// 检测数据库联机是否成功
	if err := data.Ping(context.Background()).Err(); err != nil {
		helper.Errorf("redis数据库连接失败,失败信息:%s\n", err)
		return nil, nil, err
	}

	// 用于关闭redis连接池的函数
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
