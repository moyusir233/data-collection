package service

import (
	"context"
	"fmt"
	pb "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/biz"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"io"
	"time"
)

const CLIENT_ID_HEADER = "x-client-id"

type ConfigService struct {
	pb.UnimplementedConfigServer
	uc      *biz.ConfigUsecase
	manager *biz.RouteManager
	logger  *log.Helper
}

func NewConfigService(uc *biz.ConfigUsecase, r *biz.RouteManager, logger log.Logger) (*ConfigService, func(), error) {
	c := &ConfigService{
		uc:      uc,
		manager: r,
		logger:  log.NewHelper(logger),
	}
	// 初始化配置更新所需要的路由资源
	err := c.manager.Init()
	if err != nil {
		return nil, nil, err
	}
	return c, func() {
		if err := c.manager.Close(); err != nil {
			c.logger.Error(err)
		}
	}, nil
}

func (s *ConfigService) CreateInitialConfigSaveStream0(conn pb.Config_CreateInitialConfigSaveStream0Server) error {
	// 设备类别号，代码生成时注入
	var (
		clientID      string
		deviceClassID = 0
	)

	// 检查请求头中是否包含clientID
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	}

	for {
		config, err := conn.Recv()
		if err == io.EOF {
			return conn.SendAndClose(&pb.ConfigServiceReply{Success: true})
		} else if err != nil {
			return err
		}

		// 提取设备基本信息进行保存或路由的激活
		info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
		info.DeviceID = config.Id

		// 若clientID不为空，则建立关于该clientID的路由信息
		// TODO 考虑错误处理
		if clientID != "" {
			err = s.manager.ActivateRoute(clientID, info)
			if err != nil {
				return err
			}
		}

		// TODO 设备初始配置保存出错时如何处理，使用怎样的错误模型返回？
		if err = s.uc.SaveDeviceConfig(info, config); err != nil {
			return err
		}
	}
}

func (s *ConfigService) CreateConfigUpdateStream0(conn pb.Config_CreateConfigUpdateStream0Server) error {
	var (
		clientID      string
		updateChannel chan interface{}
		deviceClassID = 0
		info          = &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
	)

	// 首先从客户端流请求头中提取clientID，允许客户端复用clientID,
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	} else {
		// 若请求头中不存在，则分配clientID，通过响应头并发送给客户端
		// 利用uuid作为clientID，使用的uuid version1
		uid, err := uuid.NewUUID()
		if err != nil {
			clientID = fmt.Sprintf("%s%d", conf.Username, time.Now().Unix())
		} else {
			clientID = uid.String()
		}

		// 将clientID存放到响应头中发送
		md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
		err = conn.SendHeader(md)
		// TODO 考虑错误处理
		if err != nil {
			return err
		}
	}

	// 获得clientID对应的updateChannel
	updateChannel = s.manager.LoadOrCreateParentNode(clientID).UpdateChannel

	// 不断从相应的channel中获得配置更新信息，并发送给客户端
	/* TODO 修改推送配置更新消息的实现逻辑，要确保客户端发送了关闭连接的消息后，协程即立刻结束，而不是继续保持读取
	   updateChannel的状态，这样会导致用户下次发送更新请求时，会有单个配置更新消息发送被已经该结束的协程处理掉
	*/

	for c := range updateChannel {
		config := c.(*pb.DeviceConfig0)
		err := conn.Send(config)
		if err != nil {
			// TODO 考虑发送失败时，是否需要将配置更新消息存储回channel
			updateChannel <- config
			return err
		}
		reply, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// 当客户端给出发送不成功的答复时，尝试重发一次
		// TODO 考虑配置最大重发次数?
		if !reply.Success {
			// 当发送不成功，将更新信息放回channel，至下一次循环进行处理
			updateChannel <- config
		} else {
			// TODO 考虑设备配置保存失败时如何处理
			info.DeviceID = config.Id
			err := s.uc.SaveDeviceConfig(info, config)
			if err != nil {
				return err
			}
		}
		// 客户端表示需要断开连接
		if reply.End {
			break
		}
	}
	return nil
}

func (s *ConfigService) UpdateDeviceConfig0(ctx context.Context, req *pb.DeviceConfig0) (*pb.ConfigServiceReply, error) {
	// 设备类别号，代码生成时注入
	deviceClassID := 0
	info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID, DeviceID: req.Id}
	// 查询节点，将配置更新信息发送到相应channel中
	if channel, ok := s.manager.GetDeviceUpdateChannel(info); ok {
		channel <- req
		return &pb.ConfigServiceReply{Success: true}, nil
	} else {
		return nil, errors.New(400,
			"unable to find the config update stream",
			"Please confirm that the client has successfully established the configuration update flow before sending the update request",
		)
	}
}

func (s *ConfigService) CreateInitialConfigSaveStream1(conn pb.Config_CreateInitialConfigSaveStream1Server) error {
	// 设备类别号，代码生成时注入
	var (
		clientID      string
		deviceClassID = 1
	)

	// 检查请求头中是否包含clientID
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	}

	for {
		config, err := conn.Recv()
		if err == io.EOF {
			return conn.SendAndClose(&pb.ConfigServiceReply{Success: true})
		} else if err != nil {
			return err
		}

		// 提取设备基本信息进行保存或路由的激活
		info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
		info.DeviceID = config.Id

		// 若clientID不为空，则建立关于该clientID的路由信息
		// TODO 考虑错误处理
		if clientID != "" {
			err = s.manager.ActivateRoute(clientID, info)
			if err != nil {
				return err
			}
		}

		// TODO 设备初始配置保存出错时如何处理，使用怎样的错误模型返回？
		if err = s.uc.SaveDeviceConfig(info, config); err != nil {
			return err
		}
	}
}

func (s *ConfigService) CreateConfigUpdateStream1(conn pb.Config_CreateConfigUpdateStream1Server) error {
	var (
		clientID      string
		updateChannel chan interface{}
		deviceClassID = 1
		info          = &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
	)

	// 首先从客户端流请求头中提取clientID，允许客户端复用clientID,
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	} else {
		// 若请求头中不存在，则分配clientID，通过响应头并发送给客户端
		// 利用uuid作为clientID，使用的uuid version1
		uid, err := uuid.NewUUID()
		if err != nil {
			clientID = fmt.Sprintf("%s%d", conf.Username, time.Now().Unix())
		} else {
			clientID = uid.String()
		}

		// 将clientID存放到响应头中发送
		md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
		err = conn.SendHeader(md)
		// TODO 考虑错误处理
		if err != nil {
			return err
		}
	}

	// 获得clientID对应的updateChannel
	updateChannel = s.manager.LoadOrCreateParentNode(clientID).UpdateChannel

	// 不断从相应的channel中获得配置更新信息，并发送给客户端
	/* TODO 修改推送配置更新消息的实现逻辑，要确保客户端发送了关闭连接的消息后，协程即立刻结束，而不是继续保持读取
	   updateChannel的状态，这样会导致用户下次发送更新请求时，会有单个配置更新消息发送被已经该结束的协程处理掉
	*/

	for c := range updateChannel {
		config := c.(*pb.DeviceConfig1)
		err := conn.Send(config)
		if err != nil {
			// TODO 考虑发送失败时，是否需要将配置更新消息存储回channel
			updateChannel <- config
			return err
		}
		reply, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		// 当客户端给出发送不成功的答复时，尝试重发一次
		// TODO 考虑配置最大重发次数?
		if !reply.Success {
			// 当发送不成功，将更新信息放回channel，至下一次循环进行处理
			updateChannel <- config
		} else {
			// TODO 考虑设备配置保存失败时如何处理
			info.DeviceID = config.Id
			err := s.uc.SaveDeviceConfig(info, config)
			if err != nil {
				return err
			}
		}
		// 客户端表示需要断开连接
		if reply.End {
			break
		}
	}
	return nil
}

func (s *ConfigService) UpdateDeviceConfig1(ctx context.Context, req *pb.DeviceConfig1) (*pb.ConfigServiceReply, error) {
	// 设备类别号，代码生成时注入
	deviceClassID := 1
	info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID, DeviceID: req.Id}
	// 查询节点，将配置更新信息发送到相应channel中
	if channel, ok := s.manager.GetDeviceUpdateChannel(info); ok {
		channel <- req
		return &pb.ConfigServiceReply{Success: true}, nil
	} else {
		return nil, errors.New(400,
			"unable to find the config update stream",
			"Please confirm that the client has successfully established the configuration update flow before sending the update request",
		)
	}
}
