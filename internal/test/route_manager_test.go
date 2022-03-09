package test

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/util/kong"
	"github.com/imroc/req/v3"
	"net/http"
	"os"
	"testing"
	"time"
)

func TestBiz_RouteManager(t *testing.T) {
	const KONG_PROXY_ADDRESS = "http://kong.test.svc.cluster.local:8000"

	// 测试RouteManager注册与注销网关组件的功能

	// 初始化测试所需组件
	envs := map[string]string{
		"USERNAME":           "test",
		"DEVICE_CLASS_COUNT": "5",
		"SERVICE_NAME":       "test",
		"SERVICE_HOST":       "nginx.test.svc.cluster.local",
		"APP_DOMAIN_NAME":    "kong.test.svc.cluster.local",
	}
	for k, v := range envs {
		err := os.Setenv(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	bootstrap, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	routeManager := biz.NewRouteManager(bootstrap.Server)
	info := &biz.DeviceGeneralInfo{
		DeviceClassID: 0,
		DeviceID:      "test",
	}
	getID := func(info *biz.DeviceGeneralInfo) string {
		return fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
	}

	// kong客户端和http客户端
	admin := kong.NewAdmin(bootstrap.Server.Gateway.Address)
	client := req.C().SetBaseURL(KONG_PROXY_ADDRESS).DevMode()

	// 用于测试key-auth插件的consumer和key
	consumer, err := admin.Create(&kong.ConsumerCreateOption{Username: "test"})
	if err != nil {
		t.Fatal(err)
	}
	key, err := admin.Create(&kong.KeyCreateOption{Username: "test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Delete(key)
		admin.Delete(consumer)
	})

	// 初始化网关组件，注册测试的清理函数
	err = routeManager.Init()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		routeManager.Close()
	})

	// 激活路由，测试网关组件是否正常工作
	t.Run("Test_Register_Gateway", func(t *testing.T) {
		err = routeManager.ActivateRoute("test", info)
		if err != nil {
			t.Fatal(err)
		}
		time.Sleep(time.Second)
		// 子测试结束时清除注册的路由
		t.Cleanup(func() {
			routeManager.UnRegisterRoute(info)
			time.Sleep(time.Second)
		})

		// 不携带key进行发送
		response, err := client.R().SetHeader("X-Device-ID", getID(info)).Get("")
		if err != nil {
			t.Fatal(err)
		}
		if response.StatusCode != http.StatusUnauthorized {
			t.Error("failed to register auth plugin")
			return
		}
		// 携带key进行发送
		response, err = client.R().
			SetHeaders(map[string]string{
				"X-Device-ID": getID(info),
				"X-Api-Key":   key.(*kong.Key).Key,
			}).Get("")
		if err != nil {
			t.Fatal(err)
		}
		if response.IsError() {
			t.Error("failed to register auth plugin")
		}
	})

	// 对四种激活路由的方式进行测试(这里只是简单的覆盖测试,没有对路由激活的效果进行验证)

	t.Run("Test_Activate_Route", func(t *testing.T) {
		// 第一种，传入的初始节点为空,info对应的设备信息未注册
		clientID1 := "test1"
		err = routeManager.ActivateRoute(clientID1, info)
		if err != nil {
			t.Fatal(err)
		}

		// 第二种，传入的初始节点为空,info对应的设备信息已注册,测试初始节点是否进行了复用
		clientID2 := "test2"
		err = routeManager.ActivateRoute(clientID2, info)
		if err != nil {
			t.Fatal(err)
		}
		if routeManager.LoadOrCreateParentNode(clientID1) != routeManager.LoadOrCreateParentNode(clientID2) {
			t.Error("The initial node is not multiplexed")
		}

		// 第三种，传入的初始节点不为空，info对应的设备信息未注册
		info2 := &biz.DeviceGeneralInfo{
			DeviceClassID: 1,
			DeviceID:      "test",
		}
		err = routeManager.ActivateRoute(clientID1, info2)
		if err != nil {
			t.Fatal(err)
		}

		// 第四种，传入的初始节点不为空，info对应的设备信息已注册
		info3 := &biz.DeviceGeneralInfo{
			DeviceClassID: 2,
			DeviceID:      "test",
		}
		clientID3 := "test3"
		err = routeManager.ActivateRoute(clientID3, info3)
		if err != nil {
			t.Fatal(err)
		}
		err = routeManager.ActivateRoute(clientID1, info3)
		if err != nil {
			t.Fatal(err)
		}
	})

}
