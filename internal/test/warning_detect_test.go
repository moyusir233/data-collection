package test

import (
	"context"
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/data"
	v1 "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestBiz_WarningDetectUsecase_SaveDeviceState(t *testing.T) {
	bc, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	usecase, cleanUp, err := InitWarningDetectUsecase(bc.Data, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanUp)

	info := &biz.DeviceGeneralInfo{
		Username:      t.Name(),
		DeviceClassID: 0,
		DeviceID:      t.Name(),
	}
	state := v1.TestedDeviceState{
		Id:          t.Name(),
		Voltage:     5,
		Current:     3,
		Temperature: 22,
	}
	fields := map[string]float64{
		"Voltage":     state.Voltage,
		"Temperature": state.Temperature,
	}

	// 新建测试用的redis客户端
	client, cleanUp2, err := data.NewData(bc.Data, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	// 注册关闭连接和删除测试中创建的键的清理函数
	t.Cleanup(cleanUp2)
	t.Cleanup(func() {
		client.Del(context.Background(), biz.GetDeviceStateKey(info))
		for k, _ := range fields {
			client.Del(context.Background(), biz.GetDeviceStateFieldKey(info, k))
		}
	})

	// 提前创建fields需要的ts
	for k, _ := range fields {
		client.Do(context.Background(), "TS.CREATE", biz.GetDeviceStateFieldKey(info, k))
	}

	err = usecase.SaveDeviceState(info, &state, fields)
	if err != nil {
		t.Fatal(err)
	}

	// 通过删除元素判断完整的设备状态信息是否保存成功
	marshal, err := proto.Marshal(&state)
	if err != nil {
		t.Fatal(err)
	}
	v := fmt.Sprintf("%x", marshal)
	result, err := client.ZRem(context.Background(), biz.GetDeviceStateKey(info), v).Result()
	if err != nil {
		t.Fatal(err)
	}
	if result == 0 {
		t.Fatal("Failed to save device status information")
	}

	// 通过查询每个字段对应ts的值并比较，判断保存是否成功
	for k, v := range fields {
		slice, err := client.Do(context.Background(), "TS.GET", k).Slice()
		if err != nil {
			t.Error(err)
		} else if len(slice) == 0 {
			t.Fail()
		} else {
			query, ok := slice[1].(float64)
			if !ok {
				t.Fail()
			} else if query != v {
				t.Error("The query result is inconsistent with the saved result")
			}
		}
	}
}
