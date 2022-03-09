package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"time"
)

// RedisRepo redis数据库操作对象，可以理解为dao
type RedisRepo struct {
	client *Data
	logger *log.Helper
}

// NewRedisRepo 实例化redis数据库操作对象
func NewRedisRepo(data *Data, logger log.Logger) biz.UnionRepo {
	return &RedisRepo{
		client: data,
		logger: log.NewHelper(logger),
	}
}

// SaveDeviceConfig 保存设备配置到redis的hash中
func (r *RedisRepo) SaveDeviceConfig(key, field string, value []byte) error {
	// 这里将value转换为十六进制的字符串进行保存
	v := fmt.Sprintf("%x", value)
	if err := r.client.HSet(context.Background(), key, field, v).Err(); err != nil {
		return err
	}
	return nil
}

// SaveDeviceState 以timestamp为score,保存设备状态信息到zset中,并保存设备状态预警字段至timeseries中
func (r *RedisRepo) SaveDeviceState(state *biz.DeviceState, fields ...*biz.DeviceStateField) error {
	// 利用redis事务确保设备状态信息和其预警字段信息共同保存
	_, err := r.client.TxPipelined(context.Background(), func(p redis.Pipeliner) error {
		// 将value转换为十六进制字符串进行保存
		v := fmt.Sprintf("%x", state.Value)
		p.ZAdd(context.Background(), state.Key, &redis.Z{
			Score:  float64(time.Now().Unix()),
			Member: v,
		})
		if len(fields) > 0 {
			t := time.Now().Unix()
			for _, f := range fields {
				args := make([]interface{}, 0, len(fields)*3+1)
				args = append(args, "TS.ADD")
				args = append(args, f.Key, t, f.Value)

				// 当字段对应的ts未被创建时，以下设置的参数会被用于ts的创建

				// 设置ts中数据的时间戳最大跨度,redis会根据这个值自动对ts执行trim操作
				args = append(args, "RETENTION", r.client.retention.Milliseconds())
				// 设置ts的标签
				args = append(args, "LABELS", biz.WarningDetectFieldLabelName, f.Label)
				p.Do(context.Background(), args...)
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}
