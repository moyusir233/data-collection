package test

import (
	"context"
	"errors"
	"fmt"
	v1 "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/biz"
	"gitee.com/moyusir/data-collection/internal/service"
	"github.com/go-kratos/kratos/v2/metadata"
	md "google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
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

	// 初始化测试所需环境变量
	envs := map[string]string{
		"USERNAME":           "test",
		"DEVICE_CLASS_COUNT": "5",
		"SERVICE_NAME":       "test",
		"SERVICE_HOST":       "auto-test-server.test.svc.cluster.local",
		"APP_DOMAIN_NAME":    "kong.test.svc.cluster.local",
	}
	bootstrap, err := generalInit("", envs)
	if err != nil {
		t.Fatal(err)
	}

	configClient, configHttpClient, warningDetectClient := StartDataCollectionTestServer(t, bootstrap)

	// 声明两个clientID用于测试
	var (
		clientID1, clientID2 string
	)

	// 辅助函数的声明
	var (
		createConfigUpdateStream func(t *testing.T, cid string) (stream v1.Config_CreateConfigUpdateStream1Client, clientID string, err error)
		createStateInfoStream    func(t *testing.T, cid string) (stream v1.WarningDetect_CreateStateInfoSaveStream1Client, clientID string, err error)
		createInitConfigStream   func(t *testing.T, cid string) (stream v1.Config_CreateInitialConfigSaveStream1Client, err error)
		sendUpdateConfigRequest  func(configs ...*v1.DeviceConfig1) error
		checkRecvConfig          func(stream v1.Config_CreateConfigUpdateStream1Client, configs ...*v1.DeviceConfig1) error
	)
	// 辅助函数的实现
	{
		// 创建配置更新流的辅助函数
		createConfigUpdateStream = func(t *testing.T, cid string) (stream v1.Config_CreateConfigUpdateStream1Client, clientID string, err error) {
			ctx := context.Background()

			// 若传入的cid不为空，表示需要进行clientID的复用，因此进行请求头的配置
			if cid != "" {
				ctx = md.NewOutgoingContext(
					context.Background(),
					map[string][]string{service.CLIENT_ID_HEADER: {cid}})
				clientID = cid
			}

			// 建立更新流
			stream, err = configClient.CreateConfigUpdateStream1(ctx)
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
		createStateInfoStream = func(t *testing.T, cid string) (stream v1.WarningDetect_CreateStateInfoSaveStream1Client, clientID string, err error) {
			// 配置请求头
			ctx := context.Background()
			if cid != "" {
				ctx = md.NewOutgoingContext(
					context.Background(),
					map[string][]string{service.CLIENT_ID_HEADER: {cid}})
			}

			// 创建流
			stream, err = warningDetectClient.CreateStateInfoSaveStream1(ctx)
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
		createInitConfigStream = func(t *testing.T, cid string) (stream v1.Config_CreateInitialConfigSaveStream1Client, err error) {
			// 配置请求头
			ctx := context.Background()
			if cid != "" {
				ctx = md.NewOutgoingContext(
					context.Background(),
					map[string][]string{service.CLIENT_ID_HEADER: {cid}})
			}

			stream, err = configClient.CreateInitialConfigSaveStream1(ctx)
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
		sendUpdateConfigRequest = func(configs ...*v1.DeviceConfig1) error {
			for _, c := range configs {
				id := biz.GetKey(&biz.DeviceGeneralInfo{
					DeviceClassID: 1,
					DeviceID:      c.Id,
				})
				// 配置http请求头
				clientContext := metadata.NewClientContext(
					context.Background(),
					map[string]string{"X-Device-ID": id},
				)

				reply, err := configHttpClient.UpdateDeviceConfig1(clientContext, c)
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
		checkRecvConfig = func(stream v1.Config_CreateConfigUpdateStream1Client, configs ...*v1.DeviceConfig1) error {
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
	}

	// 1. 测试通过上传设备初始配置，建立的设备配置更新路由是否可靠(clientID存在,注册新设备的分支)
	t.Run("Test_SaveInitDeviceConfig", func(t *testing.T) {
		// 设备初始配置信息
		configs := []*v1.DeviceConfig1{
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
		var updateStream v1.Config_CreateConfigUpdateStream1Client
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
		var updateStream v1.Config_CreateConfigUpdateStream1Client
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
		states := []*v1.DeviceState1{
			{
				Id:          "test4",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
				Time:        timestamppb.New(time.Now()),
			},
			{
				Id:          "test5",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
				Time:        timestamppb.New(time.Now()),
			},
			{
				Id:          "test6",
				Voltage:     1,
				Current:     2,
				Temperature: 3,
				Time:        timestamppb.New(time.Now()),
			},
		}
		configs := []*v1.DeviceConfig1{
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
		configs := []*v1.DeviceConfig1{
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
		config := &v1.DeviceConfig1{
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
		states := make([]*v1.DeviceState1, len(deviceIDs))
		for i, id := range deviceIDs {
			states[i] = &v1.DeviceState1{Id: id, Time: timestamppb.New(time.Now())}
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
		configs := make([]*v1.DeviceConfig1, len(deviceIDs))
		for i, id := range deviceIDs {
			configs[i] = &v1.DeviceConfig1{Id: id}
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

		err := sendUpdateConfigRequest(&v1.DeviceConfig1{Id: "test3"})
		if err == nil {
			t.Error("Failed to automatically unregister overtime route")
			return
		}
	})
}
