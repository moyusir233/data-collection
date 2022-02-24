package data

import (
	"context"
	"fmt"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
)

type RedisRepo struct {
	client *Data
	logger *log.Helper
}

// NewRedisRepo 实例化redis数据库操作对象
func NewRedisRepo(data *Data, logger log.Logger) *RedisRepo {
	return &RedisRepo{
		client: data,
		logger: log.NewHelper(logger),
	}
}

// SaveDataToHash 保存数据
func (r *RedisRepo) SaveDataToHash(key, field string, value []byte) error {
	v := fmt.Sprintf("%x", value)
	err := r.client.HSet(context.Background(), key, field, v).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisRepo) SaveDataToZset(key string, score float64, value []byte) error {
	v := fmt.Sprintf("%x", value)
	err := r.client.ZAdd(context.Background(), key, &redis.Z{
		Score:  score,
		Member: v,
	}).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisRepo) SaveDataToTs(key string, stamp int64, value []byte) error {
	v := fmt.Sprintf("%x", value)
	err := r.client.Do(context.Background(), "TS.ADD", key, stamp, v).Err()
	if err != nil {
		return err
	}
	return nil
}
