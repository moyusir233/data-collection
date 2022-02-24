package data

import (
	"context"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-redis/redis/v8"
	"time"
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
	err := r.client.HSet(context.Background(), key, field, value).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisRepo) SaveDataToZset(key string, score float64, value []byte) error {
	err := r.client.ZAdd(context.Background(), key, &redis.Z{
		Score:  score,
		Member: value,
	}).Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *RedisRepo) SaveDataToTs(key string, stamp time.Time, value []byte) error {
	err := r.client.Do(context.Background(), "TS.ADD", key, stamp, "test").Err()
	if err != nil {
		return err
	}
	return nil
}
