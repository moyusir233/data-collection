package test

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/util/kong"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/imroc/req/v3"
	"net/http"
	"testing"
)

func TestGatewayRegister(t *testing.T) {
	// 导入配置
	bootstrap, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	// 建立kong客户端，创建consumer和其对应的key，为后面的register测试做准备
	admin := kong.NewAdmin(bootstrap.Server.Gateway.Address, log.DefaultLogger)
	consumer, err := admin.Create(&kong.ConsumerCreateOption{Username: conf.Username})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Delete(consumer)
	})
	key, err := admin.Create(&kong.KeyCreateOption{Username: conf.Username})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Delete(key)
	})
	// 实例化register
	register := biz.NewGatewayRegister(bootstrap.Server, log.DefaultLogger)
	t.Cleanup(func() {
		register.Close()
	})
	err = register.Init()
	if err != nil {
		t.Fatal(err)
	}
	// 激活指向测试http服务器的路由
	info := &biz.DeviceGeneralInfo{
		DeviceClassID: 0,
		DeviceID:      "test",
	}
	deviceID := fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
	err = register.ActivateRoute(info)
	if err != nil {
		t.Fatal(err)
	}
	// 发出请求，一次不携带api-key，一次携带
	client := req.C().SetBaseURL("http://kong.test.svc.cluster.local:8000")
	response, err := client.R().SetHeader("X-Device-ID", deviceID).Get("")
	if err != nil {
		t.Error(err)
	}
	if response.StatusCode != http.StatusUnauthorized {
		t.Error("Auth plugin does not work")
	}

	response, err = client.R().SetHeaders(map[string]string{
		"X-Device-ID": deviceID,
		"X-Api-Key":   key.(*kong.Key).Key,
	}).Get("")
	if err != nil {
		t.Error(err)
	}
	if response.IsError() {
		t.Error("Auth plugin does not work")
	}
	// 注销掉route，再发出请求
	register.UnRegisterRoute(info)
	response, err = client.R().SetHeaders(map[string]string{
		"X-Device-ID": deviceID,
		"X-Api-Key":   key.(*kong.Key).Key,
	}).Get("")
	if err != nil {
		t.Error(err)
	}
	if !response.IsError() {
		t.Error("route unregister failed")
	}
}
