package service

import (
	pb "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/biz"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/grpc/metadata"
	"io"
)

type WarningDetectService struct {
	pb.UnimplementedWarningDetectServer
	uc      *biz.WarningDetectUsecase
	updater *biz.DeviceConfigUpdater
	logger  *log.Helper
}

func NewWarningDetectService(uc *biz.WarningDetectUsecase, updater *biz.DeviceConfigUpdater, logger log.Logger) *WarningDetectService {
	return &WarningDetectService{
		uc:      uc,
		updater: updater,
		logger:  log.NewHelper(logger),
	}
}

func (s *WarningDetectService) CreateStateInfoSaveStream0(conn pb.WarningDetect_CreateStateInfoSaveStream0Server) error {
	var (
		clientID string
		// 设备类别号，代码生成时注入
		deviceClassID = 0
		// 设备预警字段，代码生成时注入
		fields = map[string]float64{
			"Voltage": 0,
			"Current": 0,
		}
		// 设备非时间，非id的字段，代码生成时注入
		tags = map[string]string{}
	)

	// 检查请求头中是否包含clientID
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
				500, "Service_State_Error", "发送grpc请求头时发生了错误:%v", err)
		}
	}

	// 将clientID存放到响应头中发送
	md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
	// TODO 考虑错误处理
	if err := conn.SendHeader(md); err != nil {
		return errors.Newf(
			500, "Service_State_Error", "发送grpc请求头时发生了错误:%v", err)
	}

	s.logger.Infof("与 %v 建立了传输设备状态信息的grpc流", clientID)

	for {
		state, err := conn.Recv()
		if err == io.EOF {
			s.logger.Infof("关闭了 %v 的传输设备状态信息的grpc流", clientID)
			return nil
		}
		if err != nil {
			return errors.Newf(
				500, "Service_State_Error",
				"接收用户 %v 传输的设备状态信息时发生了错误:%v", clientID, err)
		}

		// 提取设备状态信息进行路由激活以及保存
		info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
		info.DeviceID = state.Id
		// TODO 这里不应该使用反射去提取值，而是通过代码生成提取
		fields["Voltage"] = state.Voltage
		fields["Current"] = state.Current

		// TODO 考虑路由激活以及保存设备状态出错时如何处理
		err = s.updater.ConnectDeviceAndClientID(clientID, info)
		if err != nil {
			return err
		}
		err = s.uc.SaveDeviceState(info, state.Time.AsTime(), fields, tags)
		if err != nil {
			return err
		}

		err = conn.Send(&pb.WarningDetectServiceReply{Success: true})
		if err != nil {
			return errors.Newf(
				500, "Service_State_Error",
				"向用户 %v 发送传输设备状态的响应信息时发生了错误:%v", clientID, err)
		}
	}
}

func (s *WarningDetectService) CreateStateInfoSaveStream1(conn pb.WarningDetect_CreateStateInfoSaveStream1Server) error {
	var (
		clientID string
		// 设备类别号，代码生成时注入
		deviceClassID = 1
		// 设备预警字段，代码生成时注入
		fields = map[string]float64{
			"Voltage": 0,
			"Current": 0,
		}
		// 设备非时间，非id的字段，代码生成时注入
		tags = map[string]string{}
	)

	// 检查请求头中是否包含clientID
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
				500, "Service_State_Error", "发送grpc请求头时发生了错误:%v", err)
		}
	}

	// 将clientID存放到响应头中发送
	md = metadata.New(map[string]string{CLIENT_ID_HEADER: clientID})
	// TODO 考虑错误处理
	if err := conn.SendHeader(md); err != nil {
		return err
	}

	for {
		state, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		// 提取设备状态信息进行路由激活以及保存
		info := &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
		info.DeviceID = state.Id
		// TODO 这里不应该使用反射去提取值，而是通过代码生成提取
		fields["Voltage"] = state.Voltage
		fields["Current"] = state.Current

		// TODO 考虑路由激活以及保存设备状态出错时如何处理
		err = s.updater.ConnectDeviceAndClientID(clientID, info)
		if err != nil {
			return err
		}
		err = s.uc.SaveDeviceState(info, state.Time.AsTime(), fields, tags)
		if err != nil {
			return err
		}

		err = conn.Send(&pb.WarningDetectServiceReply{Success: true})
		if err != nil {
			return err
		}
	}
}
