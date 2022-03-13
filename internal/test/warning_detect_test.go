package test

import (
	"context"
	"fmt"
	"gitee.com/moyusir/data-collection/internal/biz"
	"gitee.com/moyusir/data-collection/internal/data"
	v1 "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestBiz_WarningDetectUsecase_SaveDeviceState(t *testing.T) {
	// 初始化测试环境
	bc, err := generalInit("", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	usecase, cleanUp, err := InitWarningDetectUsecase(bc.Data, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanUp)

	// 定义用于保存的测试信息
	info := &biz.DeviceGeneralInfo{
		DeviceClassID: 0,
		DeviceID:      t.Name(),
	}
	state := v1.TestedDeviceState{
		Id:          t.Name(),
		Voltage:     5,
		Current:     3,
		Temperature: 22,
		Time:        timestamppb.New(time.Now()),
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
			key, _ := biz.GetDeviceStateFieldKeyAndLabel(info, k)
			client.Del(context.Background(), key)
		}
	})

	// 保存设备状态
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
		key, _ := biz.GetDeviceStateFieldKeyAndLabel(info, k)
		slice, err := client.Do(context.Background(), "TS.GET", key).Slice()
		if err != nil {
			t.Error(err)
		} else if len(slice) == 0 {
			t.Errorf("The TS capacity corresponding to the key %v is 0\n", key)
		} else {
			query := strings.Trim(fmt.Sprintf("%v", slice[1]), " ")
			target := strconv.FormatFloat(v, 'f', 0, 64)
			if query != target {
				t.Errorf("The value saved in TS %v is different from the value queried %v\n",
					query, target,
				)
			}
		}
	}
}
