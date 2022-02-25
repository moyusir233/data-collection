package test

import (
	"context"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestData_RedisRepo(t *testing.T) {
	// 初始化redis连接
	bc, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	client, cleanUp, err := data.NewData(bc.Data, log.NewStdLogger(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	// 注册测试完毕后执行的清理函数
	t.Cleanup(cleanUp)

	// 初始化redis dao
	redisRepo := data.NewRedisRepo(client, log.NewStdLogger(os.Stdout))
	// 允许并行运行子测试
	t.Parallel()
	// 设备保存测试
	t.Run("Save_Device_Config", func(t *testing.T) {
		// 注册清理函数，删除测试中创建的hash
		t.Cleanup(func() {
			client.Del(context.Background(), t.Name())
		})
		err := redisRepo.SaveDeviceConfig(t.Name(), t.Name(), []byte(t.Name()))
		if err != nil {
			t.Error(err)
			return
		}
		// 测试是否能够查询得到刚保存在hash中的信息
		err = client.HGet(context.Background(), t.Name(), t.Name()).Err()
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
		// 注册清理函数，清理测试中创建的zset和ts
		t.Cleanup(func() {
			client.Del(context.Background(), state.Key)
			for _, f := range fields {
				client.Del(context.Background(), f.Key)
			}
		})
		// 创建ts
		for _, f := range fields {
			err := client.Do(context.Background(), "TS.CREATE", f.Key).Err()
			if err != nil {
				t.Error(err)
				return
			}
		}
		err := redisRepo.SaveDeviceState(state, fields...)
		if err != nil {
			t.Error(err)
			return
		}
		// 通过查询操作测试上述的保存操作是否成功
		// 查询对应zset的size
		result, err := client.ZCard(context.Background(), state.Key).Result()
		if err != nil {
			t.Error(err)
		} else if result == 0 {
			t.Error("The corresponding Zset capacity is 0")
		}
		// 查询每个field对应的ts是否为空
		for _, f := range fields {
			result, err := client.Do(context.Background(), "TS.GET", f.Key).Slice()
			if err != nil {
				t.Error(err)
			} else if len(result) == 0 {
				t.Error("The corresponding Ts capacity is 0")
			}
		}
	})
}
