package test

import (
	"context"
	v1 "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/service"
	utilApi "gitee.com/moyusir/util/api/util/v1"
	"gitee.com/moyusir/util/kong"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/metadata"
	middlewareMD "github.com/go-kratos/kratos/v2/middleware/metadata"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	g "google.golang.org/grpc"
	md "google.golang.org/grpc/metadata"
	"os"
	"testing"
	"time"
)

func TestDataCollectionService(t *testing.T) {
	const (
		KONG_HTTP_ADDRESS = "kong.test.svc.cluster.local:8000"
	)
	// 初始化测试所需环境变量
	envs := map[string]string{
		"USERNAME":           "test",
		"DEVICE_CLASS_COUNT": "5",
		"SERVICE_NAME":       "test",
		"SERVICE_HOST":       "auto-test-server.test.svc.cluster.local",
		"APP_DOMAIN_NAME":    "kong.test.svc.cluster.local",
	}
	for k, v := range envs {
		err := os.Setenv(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	// 导入配置，并启动服务器
	bootstrap, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}

	logger := log.DefaultLogger
	app, cleanUp, err := initApp(bootstrap.Server, bootstrap.Data, logger)
	if err != nil {
		t.Fatal(err)
	}

	// done用来等待服务器关闭完毕
	done := make(chan struct{})
	t.Cleanup(func() {
		app.Stop()
		cleanUp()
		<-done
	})
	go func() {
		defer close(done)
		err = app.Run()
		if err != nil {
			t.Error(err)
		}
	}()

	// 等待服务器开启，然后创建测试用的grpc、http与网关客户端以及使用的api密钥
	var (
		grpcConn            *g.ClientConn
		httpConn            *http.Client
		configClient        v1.ConfigClient
		warningDetectClient v1.WarningDetectClient
		configHttpClient    v1.ConfigHTTPClient
		admin               *kong.Admin
		apiKey              *kong.Key
	)

	// 创建网关客户端，并创建api密钥
	admin = kong.NewAdmin(bootstrap.Server.Gateway.Address)
	consumer, err := admin.Create(&kong.ConsumerCreateOption{Username: "test"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Delete(consumer)
	})

	key, err := admin.Create(&kong.KeyCreateOption{Username: "test"})
	if err != nil {
		return
	}
	apiKey = key.(*kong.Key)
	t.Cleanup(func() {
		admin.Delete(key)
	})

	// 创建http与grpc的连接与客户端
initClient:
	for {
		select {
		case <-done:
			t.Fatal("server start fail")
		default:
			// 创建grpc客户端
			// 因为grpc服务的网关组件由服务中心创建，所以这里没有通过网关建立grpc连接，而是本地直接连接
			grpcConn, err = grpc.DialInsecure(
				context.Background(),
				grpc.WithEndpoint("localhost:9000"),
			)
			if err != nil {
				grpcConn.Close()
				continue
			} else {
				configClient = v1.NewConfigClient(grpcConn)
				warningDetectClient = v1.NewWarningDetectClient(grpcConn)
			}

			// 创建http客户端时，配置请求头的插件，存放密钥和路由的请求头
			httpConn, err = http.NewClient(
				context.Background(),
				http.WithEndpoint(KONG_HTTP_ADDRESS),
				http.WithMiddleware(
					middlewareMD.Client(
						middlewareMD.WithPropagatedPrefix("X-Device-ID"),
						middlewareMD.WithConstants(map[string]string{"X-Api-Key": apiKey.Key}),
					),
				),
			)

			if err != nil {
				httpConn.Close()
				continue
			} else {
				configHttpClient = v1.NewConfigHTTPClient(httpConn)
			}
			break initClient
		}
	}
	t.Cleanup(func() {
		grpcConn.Close()
		httpConn.Close()
	})

	// 声明两个clientID用于测试
	var (
		clientID1, _ string
	)

	// 1. 测试通过上传初始设备信息，建立设备配置更新路由
	t.Run("Test_SaveInitDeviceConfig", func(t *testing.T) {
		// 设备初始配置信息
		configs := []*utilApi.TestedDeviceConfig{
			{
				Id:     "test1",
				Status: false,
			},
			{
				Id:     "test2",
				Status: false,
			},
			{
				Id:     "test3",
				Status: false,
			},
		}

		// 建立更新流
		updateStream, err := configClient.CreateConfigUpdateStream(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		t.Cleanup(func() {
			updateStream.CloseSend()
		})

		// 从更新流响应头中获得clientID1
		header, err := updateStream.Header()
		if err != nil {
			t.Error(err)
			return
		}
		if tmp := header.Get(service.CLIENT_ID_HEADER); len(tmp) != 0 {
			clientID1 = tmp[0]
		} else {
			t.Error("can not find the header of clientID")
			return
		}

		// 利用获得的clientID1上传初始配置信息
		ctx := md.NewOutgoingContext(
			context.Background(),
			map[string][]string{service.CLIENT_ID_HEADER: {clientID1}},
		)
		initConfigStream, err := configClient.CreateInitialConfigSaveStream(ctx)
		if err != nil {
			t.Error(err)
			return
		}
		t.Cleanup(func() {
			initConfigStream.CloseSend()
		})

		for _, c := range configs {
			err := initConfigStream.Send(c)
			if err != nil {
				t.Error("Failed to upload the initial configs")
				return
			}
		}

		// 上传完毕后，尝试发送http请求更新配置信息
		time.Sleep(time.Second)
		for _, c := range configs {
			id := biz.GetKey(&biz.DeviceGeneralInfo{
				DeviceClassID: 0,
				DeviceID:      c.Id,
			})
			// 配置http请求头
			clientContext := metadata.NewClientContext(
				context.Background(),
				map[string]string{"X-Device-ID": id},
			)

			reply, err := configHttpClient.UpdateDeviceConfig(clientContext, c)
			if err != nil {
				t.Error(err)
				return
			}
			if !reply.Success {
				t.Error("Failed to send the request which updates the device config")
				return
			}
		}
	})

	// 2. 测试通过创建设备状态信息流，建立设备配置更新路由
	t.Run("Test_CreateStateInfoSaveStream", func(t *testing.T) {
		infoSaveStream, err := warningDetectClient.CreateStateInfoSaveStream(context.Background())
		if err != nil {
			t.Error(err)
			return
		}
		t.Cleanup(func() {
			infoSaveStream.CloseSend()
		})
	})
}
