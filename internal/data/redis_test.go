package data

import (
	"context"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
	"time"
)

func TestRedisRepo(t *testing.T) {
	c := config.New(config.WithSource(file.NewSource("../../configs/config.yaml")))
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	data, cleanUp, err := NewData(bc.Data, log.NewStdLogger(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanUp()
	redisRepo := NewRedisRepo(data, log.NewStdLogger(os.Stdout))
	t.Parallel()
	t.Run("hash_test", func(t *testing.T) {
		err := redisRepo.SaveDataToHash("hash_test", "test", []byte("test"))
		if err != nil {
			t.Error(err)
		} else {
			data.Del(context.Background(), "hash_test")
		}
	})
	t.Run("zset_test", func(t *testing.T) {
		err := redisRepo.SaveDataToZset("zset_test", 100, []byte("test"))
		if err != nil {
			t.Error(err)
		} else {
			data.Del(context.Background(), "zset_test")
		}
	})
	t.Run("ts_test", func(t *testing.T) {
		err = redisRepo.SaveDataToTs("ts_test", time.Now().Unix(), []byte("test"))
		if err != nil {
			t.Error(err)
		} else {
			data.Del(context.Background(), "ts_test")
		}
	})
}
