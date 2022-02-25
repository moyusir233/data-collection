package data

import (
	"context"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestRedisRepo(t *testing.T) {
	// 初始化redis连接
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
	// 注册测试完毕后执行的清理函数
	t.Cleanup(cleanUp)

	// 初始化redis dao
	redisRepo := NewRedisRepo(data, log.NewStdLogger(os.Stdout))
	// 允许并行运行子测试
	t.Parallel()
	// 设备保存测试
	t.Run("Save_Device_Config", func(t *testing.T) {
		err := redisRepo.SaveDeviceConfig(t.Name(), t.Name(), []byte(t.Name()))
		if err != nil {
			t.Error(err)
			return
		}
		// 若保存成功，注册清理函数，删除测试中创建的hash
		t.Cleanup(func() {
			data.Del(context.Background(), t.Name())
		})
		// 测试是否能够查询得到刚保存在hash中的信息
		err = data.HGet(context.Background(), t.Name(), t.Name()).Err()
		if err != nil {
			t.Error(err)
		}
	})
	// 设备状态信息保存测试
	t.Run("Save_Device_State", func(t *testing.T) {
		var (
			state = &biz.DeviceState{
				Key:   t.Name(),
				Value: []byte(t.Name()),
			}
			fields = []*biz.DeviceStateField{
				{
					Key:   t.Name() + "_Field1",
					Value: 1,
				},
				{
					Key:   t.Name() + "_Field2",
					Value: 2,
				},
			}
		)
		err := redisRepo.SaveDeviceState(state, fields...)
		if err != nil {
			t.Error(err)
			return
		}
		// 注册清理函数，清理测试中创建的zset和ts
		t.Cleanup(func() {
			data.Del(context.Background(), state.Key)
			for _, f := range fields {
				data.Del(context.Background(), f.Key)
			}
		})
		// 通过查询操作测试上述的保存操作是否成功
		result, err := data.ZCard(context.Background(), state.Key).Result()
		if err != nil {
			t.Error(err)
		} else if result == 0 {
			t.Fail()
		}
		for _, f := range fields {
			result, err := data.Do(context.Background(), "TS.GET", f.Key).Slice()
			if err != nil {
				t.Error(err)
			} else if len(result) == 0 {
				t.Fail()
			}
		}
	})
	t.Logf("%s测试成功", t.Name())
}
