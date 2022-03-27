package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-redis/redis/v8"
	"github.com/google/wire"
	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
)

// ProviderSet is data providers.
var ProviderSet = wire.NewSet(NewRedisData, NewInfluxdbData, NewRepo)

// RedisData 连接redis的客户端
type RedisData struct {
	// redis连接客户端
	*redis.ClusterClient
}

// InfluxdbData 连接influxdb的客户端
type InfluxdbData struct {
	// influxdb连接的客户端
	influxdb2.Client
	// 用户的组织信息
	org string
}

// NewRedisData 实例化redis数据库连接对象
func NewRedisData(c *conf.Data) (*RedisData, func(), error) {
	data := new(RedisData)

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

	// 检测数据库联机是否成功
	if err := data.Ping(context.Background()).Err(); err != nil {
		return nil, nil, err
	}

	// 用于关闭redis连接池的函数
	cleanup := func() {
		data.Close()
	}

	return data, cleanup, nil
}

func NewInfluxdbData(data *conf.Data) (*InfluxdbData, func(), error) {
	client := influxdb2.NewClient(data.Influxdb.ServerUrl, data.Influxdb.AuthToken)
	ping, err := client.Ping(context.Background())
	if err != nil {
		client.Close()
		return nil, nil, err
	} else if !ping {
		client.Close()
		return nil, nil, errors.New(
			500, "influxdb connect failed", "failed to connect the influxdb")
	}

	return &InfluxdbData{
		Client: client,
		org:    data.Influxdb.Org,
	}, func() { client.Close() }, nil
}
