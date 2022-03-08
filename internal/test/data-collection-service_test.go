package test

import (
	"context"
	"errors"
	"fmt"
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
	"google.golang.org/protobuf/proto"
	"os"
	"testing"
	"time"
)

func TestDataCollectionService(t *testing.T) {
	// 进行的测试包括:
	// 1. 测试通过上传设备初始配置，建立的设备配置更新路由是否可靠(clientID存在,注册新设备的分支)
	// 2. 测试通过上传设备状态信息，建立的设备配置更新路由是否可靠(clientID存在,注册新设备的分支)
	// 3. 测试模拟出现客户端掉线时，利用新得到的clientID上传原先的设备信息(clientID存在，新设备路由更新的分支)
	// 4. 测试不使用clientID建立设备传输连接,然后上传之前未上传过的设备信息
	//    (此时服务器会创建一个未注册的clientID进行路由信息的配置，因此对应clientID不存在，设备信息未注册的分支)
	// 5. 测试不使用clientID建立设备传输连接,然后上传之前上传过的设备信息
	//    (与上述情况类似，不过此时会为不存在的clientID复用之前的父节点，对应着clientID不存在，设备信息已注册的分支)
	// 6. 测试路由自动注销功能

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

	logger := log.NewStdLogger(os.Stdout)
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

			// 创建http客户端时，为了通过网关进行转发，需要配置请求头的插件，存放密钥和路由的请求头
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
		clientID1, clientID2 string
	)

	// 辅助函数
	var (
		createConfigUpdateStream func(t *testing.T, cid string) (stream v1.Config_CreateConfigUpdateStreamClient, clientID string, err error)
		createStateInfoStream    func(t *testing.T, cid string) (stream v1.WarningDetect_CreateStateInfoSaveStreamClient, clientID string, err error)
		createInitConfigStream   func(t *testing.T, cid string) (stream v1.Config_CreateInitialConfigSaveStreamClient, err error)
		sendUpdateConfigRequest  func(configs ...*utilApi.TestedDeviceConfig) error
		checkRecvConfig          func(stream v1.Config_CreateConfigUpdateStreamClient, configs ...*utilApi.TestedDeviceConfig) error
	)

	// 创建配置更新流的辅助函数
	createConfigUpdateStream = func(t *testing.T, cid string) (stream v1.Config_CreateConfigUpdateStreamClient, clientID string, err error) {
		ctx := context.Background()

		// 若传入的cid不为空，表示需要进行clientID的复用，因此进行请求头的配置
		if cid != "" {
			ctx = md.NewOutgoingContext(
				context.Background(),
				map[string][]string{service.CLIENT_ID_HEADER: {cid}})
			clientID = cid
		}

		// 建立更新流
		stream, err = configClient.CreateConfigUpdateStream(ctx)
		if err != nil {
			return nil, "", err
		}

		// 当传入的cid为空时，需要从响应头中获得服务器创建的clientID
		if cid == "" {
			// 从更新流响应头中获得clientID
			header, err := stream.Header()
			if err != nil {
				return nil, "", err
			}
			if tmp := header.Get(service.CLIENT_ID_HEADER); len(tmp) != 0 {
				clientID = tmp[0]
			} else {
				return nil, "", errors.New("can not find the header of clientID")
			}
		}
		return
	}

	// 创建设备状态流的辅助函数
	createStateInfoStream = func(t *testing.T, cid string) (stream v1.WarningDetect_CreateStateInfoSaveStreamClient, clientID string, err error) {
		// 配置请求头
		ctx := context.Background()
		if cid != "" {
			ctx = md.NewOutgoingContext(
				context.Background(),
				map[string][]string{service.CLIENT_ID_HEADER: {cid}})
		}

		// 创建流
		stream, err = warningDetectClient.CreateStateInfoSaveStream(ctx)
		if err != nil {
			return nil, "", err
		}
		t.Cleanup(func() {
			// 关闭客户端发送流，并接收reply，判断初始设备配置信息是否传输成功
			stream.CloseSend()
		})

		// 从stream的响应头中提取本次建立数据流使用clientID
		header, err := stream.Header()
		if err != nil {
			return nil, "", err
		}
		if tmp := header.Get(service.CLIENT_ID_HEADER); len(tmp) == 0 {
			return nil, "", errors.New("can not find the header of clientID")
		} else {
			clientID = tmp[0]
		}
		return
	}

	// 创建初始配置流的辅助函数
	createInitConfigStream = func(t *testing.T, cid string) (stream v1.Config_CreateInitialConfigSaveStreamClient, err error) {
		// 配置请求头
		ctx := context.Background()
		if cid != "" {
			ctx = md.NewOutgoingContext(
				context.Background(),
				map[string][]string{service.CLIENT_ID_HEADER: {cid}})
		}

		stream, err = configClient.CreateInitialConfigSaveStream(ctx)
		if err != nil {
			return nil, err
		}
		t.Cleanup(func() {
			reply, err := stream.CloseAndRecv()
			if err != nil {
				t.Error(err)
				return
			}
			if !reply.Success {
				err = errors.New("failed to send the initial config of device")
			}
		})
		return
	}

	// 发送配置更新http请求的辅助函数
	sendUpdateConfigRequest = func(configs ...*utilApi.TestedDeviceConfig) error {
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
				return err
			}
			if !reply.Success {
				return errors.New("failed to send the request which updates the device config")
			}
		}
		return nil
	}

	// 检查配置更新流推送的更新消息是否正确的辅助函数
	checkRecvConfig = func(stream v1.Config_CreateConfigUpdateStreamClient, configs ...*utilApi.TestedDeviceConfig) error {
		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		for i, c := range configs {
			config, err := stream.Recv()
			if err != nil {
				return err
			}

			// 发送答复，告知服务端接收成功
			// 当发送最后一条信息时，告诉服务端可以结束连接
			reply := &v1.ConfigUpdateReply{Success: true, End: false}
			if i == len(configs)-1 {
				reply.End = true
			}
			err = stream.Send(reply)
			if err != nil {
				return err
			}

			if !proto.Equal(c, config) {
				msg := fmt.Sprintf(
					"the config update information sent by the HTTP request was not pushed correctly: %v %v",
					*c, *config,
				)
				return errors.New(msg)
			}
		}
		return nil
	}

	// 1. 测试通过上传设备初始配置，建立的设备配置更新路由是否可靠(clientID存在,注册新设备的分支)
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

		// 建立更新流，获得clientID
		var updateStream v1.Config_CreateConfigUpdateStreamClient
		updateStream, clientID1, err = createConfigUpdateStream(t, "")
		if err != nil {
			t.Error(err)
			return
		}

		// 利用获得的clientID1创建流并上传初始配置信息
		initConfigStream, err := createInitConfigStream(t, clientID1)
		if err != nil {
			t.Error(err)
			return
		}

		for _, c := range configs {
			err := initConfigStream.Send(c)
			if err != nil {
				t.Error("Failed to upload the initial configs")
				return
			}
		}

		// 上传完毕后，尝试发送http请求更新配置信息
		// 休眠一段时间，确保路由组件已注册完毕再发送请求
		time.Sleep(time.Second)
		err = sendUpdateConfigRequest(configs...)
		if err != nil {
			t.Error(err)
			return
		}

		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		if err := checkRecvConfig(updateStream, configs...); err != nil {
			t.Error(err)
			return
		}
	})

	// 2. 测试通过上传设备状态信息，建立的设备配置更新路由是否可靠(clientID存在,注册新设备的分支)
	t.Run("Test_CreateStateInfoSaveStream", func(t *testing.T) {
		// 创建配置更新流，获取clientID
		var updateStream v1.Config_CreateConfigUpdateStreamClient
		updateStream, clientID2, err = createConfigUpdateStream(t, "")
		if err != nil {
			t.Error(err)
			return
		}

		// 使用获得的clientID创建设备信息传输流
		stateSaveStream, _, err := createStateInfoStream(t, clientID2)
		if err != nil {
			t.Error(err)
			return
		}

		// 定义需要传输的设备状态信息以及相对应的配置更新信息
		// 需要与第一次子测试中配置的信息区分开，避免路由的重用
		states := []*utilApi.TestedDeviceState{
			{
				Id:          "test4",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
			},
			{
				Id:          "test5",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
			},
			{
				Id:          "test6",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
			},
		}
		configs := []*utilApi.TestedDeviceConfig{
			{
				Id:     "test4",
				Status: false,
			},
			{
				Id:     "test5",
				Status: false,
			},
			{
				Id:     "test6",
				Status: false,
			},
		}

		// 通过设备状态流传输状态信息
		for _, s := range states {
			err := stateSaveStream.Send(s)
			if err != nil {
				t.Error(err)
				return
			}

			// 接收答复信息，判断是否出现错误
			reply, err := stateSaveStream.Recv()
			if err != nil {
				t.Error(err)
				return
			}
			if !reply.Success {
				t.Error(errors.New("failed to send device state info"))
				return
			}
		}

		// 上传完毕后，尝试发送http请求更新配置信息
		// 休眠一段时间，确保路由组件已注册完毕再发送请求
		time.Sleep(time.Second)
		err = sendUpdateConfigRequest(configs...)
		if err != nil {
			t.Error(err)
			return
		}

		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		if err := checkRecvConfig(updateStream, configs...); err != nil {
			t.Error(err)
			return
		}
	})

	// 由于后续测试建立在前两个测试的基础上，因此前两子测试若失败，直接结束本次测试
	if t.Failed() {
		t.FailNow()
	}

	// 3. 测试模拟出现客户端掉线时，利用新得到的clientID上传原先的设备信息(clientID存在，新设备路由更新的分支)
	t.Run("Test_UpdateRouteInfoByChangeClientID", func(t *testing.T) {
		// 创建配置更新流，获得clientID
		newUpdateStream, clientID, err := createConfigUpdateStream(t, "")
		if err != nil {
			t.Error(err)
			return
		}

		// 使用获得的clientID上传配置信息
		initConfigStream, err := createInitConfigStream(t, clientID)
		if err != nil {
			t.Error(err)
			return
		}

		// 使用测试1中上传过的配置信息进行重新上传，
		// 将test1和test2设备迁移至新clientID的路由上，而test3设备仍保持在clientID1对应的路由上
		configs := []*utilApi.TestedDeviceConfig{
			{
				Id:     "test1",
				Status: false,
			},
			{
				Id:     "test2",
				Status: false,
			},
		}

		for _, c := range configs {
			err := initConfigStream.Send(c)
			if err != nil {
				return
			}
		}

		// 上传完毕后，尝试发送http请求更新配置信息
		// 休眠一段时间，确保路由组件已更新完毕再发送请求
		time.Sleep(time.Second)
		err = sendUpdateConfigRequest(configs...)
		if err != nil {
			t.Error(err)
			return
		}

		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		if err := checkRecvConfig(newUpdateStream, configs...); err != nil {
			t.Error(err)
			return
		}

		// 测试在路由更新后，原来clientID1的路由是否还能正常使用

		// 创建clientID1对应的配置更新流
		oldUpdateStream, _, err := createConfigUpdateStream(t, clientID1)
		if err != nil {
			t.Error(err)
			return
		}

		// 发送关于test3的配置更新请求
		config := &utilApi.TestedDeviceConfig{
			Id:     "test3",
			Status: false,
		}
		err = sendUpdateConfigRequest(config)
		if err != nil {
			t.Error(err)
			return
		}

		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		if err := checkRecvConfig(oldUpdateStream, config); err != nil {
			t.Error(err)
			return
		}
	})

	// 由于第四个子测试与第五个子测试类似，这里定义一个辅助函数

	testUseUnknownClientID := func(t *testing.T, deviceIDs ...string) {
		// 直接创建设备状态数据流,获得服务端创建的clientID
		stateSaveStream, clientID, err := createStateInfoStream(t, "")
		if err != nil {
			t.Error(err)
			return
		}

		// 定义之前测试没有传输过的设备信息进行传输
		states := make([]*utilApi.TestedDeviceState, len(deviceIDs))
		for i, id := range deviceIDs {
			states[i] = &utilApi.TestedDeviceState{Id: id}
		}

		for _, s := range states {
			err := stateSaveStream.Send(s)
			if err != nil {
				t.Error(err)
				return
			}

			// 接收答复信息，判断是否出现错误
			reply, err := stateSaveStream.Recv()
			if err != nil {
				t.Error(err)
				return
			}
			if !reply.Success {
				t.Error(errors.New("failed to send device state info"))
				return
			}
		}

		// 然后通过发送http请求以及建立设备更新流接收配置更新消息，验证路由配置是否有效
		configs := make([]*utilApi.TestedDeviceConfig, len(deviceIDs))
		for i, id := range deviceIDs {
			configs[i] = &utilApi.TestedDeviceConfig{Id: id}
		}

		// 等待组件注册
		time.Sleep(time.Second)
		if err := sendUpdateConfigRequest(configs...); err != nil {
			t.Error(err)
			return
		}

		updateStream, _, err := createConfigUpdateStream(t, clientID)
		if err != nil {
			t.Error(err)
			return
		}

		// 检查更新流中接收到的配置更新消息与http发送的是否一致
		if err := checkRecvConfig(updateStream, configs...); err != nil {
			t.Error(err)
			return
		}
	}

	// 4. 测试不使用clientID建立设备传输连接,然后上传之前未上传过的设备信息
	//    (此时服务器会创建一个未注册的clientID进行路由信息的配置，因此对应clientID不存在，设备信息未注册的分支)
	t.Run("Test_UseUnknownClientIDSendUnKnownDevice", func(t *testing.T) {
		testUseUnknownClientID(t, "test7", "test8")
	})

	// 5. 测试不使用clientID建立设备传输连接,然后上传之前上传过的设备信息
	//    (与上述情况类似，不过此时会为不存在的clientID复用之前的父节点，对应着clientID不存在，设备信息已注册的分支)
	t.Run("Test_UseUnknownClientIDSendDevice", func(t *testing.T) {
		testUseUnknownClientID(t, "test4", "test5", "test6")
	})

	// 6. 测试路由自动注销功能
	t.Run("Test_AutoUnregisterRoute", func(t *testing.T) {
		// 之前测试中clientID1仍保存着test3设备的路由信息
		// 这里拿来测试路由自动注销
		// 等待路由自动注销触发
		time.Sleep(bootstrap.Server.Gateway.RouteTimeout.AsDuration())

		err := sendUpdateConfigRequest(&utilApi.TestedDeviceConfig{Id: "test3"})
		if err == nil {
			t.Error("Failed to automatically unregister overtime route")
			return
		}
	})
}
