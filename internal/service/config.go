package service

import (
	"context"
	pb "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/biz"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/metadata"
	"io"
)

const CLIENT_ID_HEADER = "x-client-id"

type ConfigService struct {
	pb.UnimplementedConfigServer
	uc      *biz.ConfigUsecase
	updater *biz.DeviceConfigUpdater
	logger  *log.Helper
}

func NewConfigService(uc *biz.ConfigUsecase, updater *biz.DeviceConfigUpdater, logger log.Logger) (*ConfigService, error) {
	c := &ConfigService{
		uc:      uc,
		updater: updater,
		logger:  log.NewHelper(logger),
	}

	return c, nil
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
		s.logger.Infof("与 %v 建立了传输设备初始配置的grpc流", clientID)
	} else {
		s.logger.Info("与未知用户建立了传输设备初始配置的grpc流")
	}

	for {
		var (
			config *pb.DeviceConfig0
			err    error
		)
		recvCtx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			config, err = conn.Recv()
		}()

		select {
		case <-conn.Context().Done():
			s.logger.Infof("关闭了 %v 的传输设备初始配置的grpc流", clientID)
			return nil
		case <-recvCtx.Done():
			if err == io.EOF {
				s.logger.Infof("关闭了 %v 的传输设备初始配置的grpc流", clientID)
				return conn.SendAndClose(&pb.ConfigServiceReply{Success: true})
			} else if err != nil {
				return errors.Newf(
					500, "Service_Config_Error",
					"接收用户 %v 的初始设备配置信息时发生了错误:%v", clientID, err)
			}

			// 提取设备基本信息进行保存或路由的激活
			info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
			info.DeviceID = config.Id

			// 若clientID不为空，则建立关于该clientID的路由信息
			// TODO 考虑错误处理
			if clientID != "" {
				err = s.updater.ConnectDeviceAndClientID(clientID, info)
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
}

func (s *ConfigService) CreateConfigUpdateStream0(conn pb.Config_CreateConfigUpdateStream0Server) error {
	var (
		clientID      string
		deviceClassID = 0
		info          = &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
	)

	// 首先从客户端流请求头中提取clientID，允许客户端复用clientID,
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	} else {
		// 若请求头中不存在，则申请创建新的clientID，通过响应头并发送给客户端
		id, err := s.updater.CreateClientID()
		if err != nil {
			return err
		} else {
			clientID = id
		}

		// 将clientID存放到响应头中发送
		md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
		err = conn.SendHeader(md)
		// TODO 考虑错误处理
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error", "发送grpc请求头时发生了错误:%v", err)
		}
	}

	s.logger.Infof("与 %v 建立了传输配置更新信息的grpc流", clientID)

	// 获得clientID对应的updateChannel
	ctx, cancel := context.WithCancel(conn.Context())
	updateChannel, err := s.updater.GetDeviceUpdateMsgChannel(ctx, clientID, new(pb.DeviceConfig0))
	if err != nil {
		return err
	}
	defer cancel()

	// 不断从相应的channel中获得配置更新信息，并发送给客户端
	for c := range updateChannel {
		config := c.(*pb.DeviceConfig0)
		err := conn.Send(config)
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error",
				"向用户 %v 发送配置更新消息时发生了错误:%v", clientID, err)
		}

		reply, err := conn.Recv()
		if err == io.EOF {
			s.logger.Infof("关闭了 %v 的传输配置更新信息的grpc流", clientID)
			return nil
		}
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error",
				"接收用户 %v 传输的配置更新消息响应时发生了错误:%v", clientID, err)
		}
		// 当客户端给出发送不成功的答复时，尝试重发一次
		// TODO 考虑配置最大重发次数?
		if !reply.Success {
			conn.Send(config)
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
			s.logger.Infof("关闭了 %v 的传输配置更新信息的grpc流", clientID)
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
	err := s.updater.UpdateDeviceConfig(info, req)
	if err != nil {
		return nil, errors.Newf(500,
			"Service_Config_Error",
			"更新设备配置时发生了未知错误:%v", err,
		)
	}

	return &pb.ConfigServiceReply{Success: true}, nil
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
		s.logger.Infof("与 %v 建立了传输设备初始配置的grpc流", clientID)
	} else {
		s.logger.Info("与未知用户建立了传输设备初始配置的grpc流")
	}

	for {
		var (
			config *pb.DeviceConfig1
			err    error
		)
		recvCtx, cancel := context.WithCancel(context.Background())
		go func() {
			defer cancel()
			config, err = conn.Recv()
		}()

		select {
		case <-conn.Context().Done():
			s.logger.Infof("关闭了 %v 的传输设备初始配置的grpc流", clientID)
			return nil
		case <-recvCtx.Done():
			if err == io.EOF {
				s.logger.Infof("关闭了 %v 的传输设备初始配置的grpc流", clientID)
				return conn.SendAndClose(&pb.ConfigServiceReply{Success: true})
			} else if err != nil {
				return errors.Newf(
					500, "Service_Config_Error",
					"接收用户 %v 的初始设备配置信息时发生了错误:%v", clientID, err)
			}

			// 提取设备基本信息进行保存或路由的激活
			info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
			info.DeviceID = config.Id

			// 若clientID不为空，则建立关于该clientID的路由信息
			// TODO 考虑错误处理
			if clientID != "" {
				err = s.updater.ConnectDeviceAndClientID(clientID, info)
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
}

func (s *ConfigService) CreateConfigUpdateStream1(conn pb.Config_CreateConfigUpdateStream1Server) error {
	var (
		clientID      string
		deviceClassID = 1
		info          = &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
	)

	// 首先从客户端流请求头中提取clientID，允许客户端复用clientID,
	md, ok := metadata.FromIncomingContext(conn.Context())
	if value := md.Get(CLIENT_ID_HEADER); ok && len(value) != 0 {
		clientID = value[0]
	} else {
		// 若请求头中不存在，则申请创建新的clientID，通过响应头并发送给客户端
		id, err := s.updater.CreateClientID()
		if err != nil {
			return err
		} else {
			clientID = id
		}

		// 将clientID存放到响应头中发送
		md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
		err = conn.SendHeader(md)
		// TODO 考虑错误处理
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error", "发送grpc请求头时发生了错误:%v", err)
		}
	}

	s.logger.Infof("与 %v 建立了传输配置更新信息的grpc流", clientID)

	// 获得clientID对应的updateChannel
	ctx, cancel := context.WithCancel(conn.Context())
	updateChannel, err := s.updater.GetDeviceUpdateMsgChannel(ctx, clientID, new(pb.DeviceConfig1))
	if err != nil {
		return err
	}
	defer cancel()

	// 不断从相应的channel中获得配置更新信息，并发送给客户端
	for c := range updateChannel {
		config := c.(*pb.DeviceConfig1)
		err := conn.Send(config)
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error",
				"向用户 %v 发送配置更新消息时发生了错误:%v", clientID, err)
		}

		reply, err := conn.Recv()
		if err == io.EOF {
			s.logger.Infof("关闭了 %v 的传输配置更新信息的grpc流", clientID)
			return nil
		}
		if err != nil {
			return errors.Newf(
				500, "Service_Config_Error",
				"接收用户 %v 传输的配置更新消息响应时发生了错误:%v", clientID, err)
		}
		// 当客户端给出发送不成功的答复时，尝试重发一次
		// TODO 考虑配置最大重发次数?
		if !reply.Success {
			conn.Send(config)
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
			s.logger.Infof("关闭了 %v 的传输配置更新信息的grpc流", clientID)
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
	err := s.updater.UpdateDeviceConfig(info, req)
	if err != nil {
		return nil, errors.Newf(500,
			"Service_Config_Error",
			"更新设备配置时发生了未知错误:%v", err,
		)
	}

	return &pb.ConfigServiceReply{Success: true}, nil
}
