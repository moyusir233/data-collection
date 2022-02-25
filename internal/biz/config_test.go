package biz

import (
	"context"
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/data"
	"gitee.com/moyusir/dataCollection/internal/test"
	v1 "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestBiz_ConfigUsecase_SaveDeviceConfig(t *testing.T) {
	bc, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	usecase, cleanUp, err := test.InitConfigUsecase(bc.Data, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanUp)
	info := &DeviceGeneralInfo{
		Username:      t.Name(),
		DeviceClassID: 0,
		DeviceID:      t.Name(),
	}
	config := v1.TestedDeviceConfig{
		Id:     t.Name(),
		Status: true,
	}
	err = usecase.SaveDeviceConfig(info, &config)
	if err != nil {
		t.Fatal(err)
	}

	client, cleanUp2, err := data.NewData(bc.Data, log.DefaultLogger)
	if err != nil {
		t.Fatal(err)
	}
	// 注册关闭连接和删除测试中创建的键的清理函数
	t.Cleanup(cleanUp2)
	t.Cleanup(func() {
		client.Del(context.Background(), getDeviceConfigKey(info))
	})

	// 通过将刚保存的信息查询出来，并比较，判断是否保存成功
	result, err := client.HGet(context.Background(), getDeviceConfigKey(info), info.DeviceID).Result()
	if err != nil {
		t.Fatal(err)
	}
	queryConfig := new(v1.TestedDeviceConfig)
	var bytes []byte
	fmt.Sscanf(result, "%x", &bytes)
	err = proto.Unmarshal(bytes, queryConfig)
	if err != nil {
		t.Fatal(err)
	}
	if !proto.Equal(&config, queryConfig) {
		t.Fatal("The query result is inconsistent with the saved result")
	}
}
