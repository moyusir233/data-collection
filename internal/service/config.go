package service

import (
	"context"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"io"

	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
)

// 定义了当前程序服务的用户及其设备的基本信息，由服务中心生成代码时注入
const (
	Username         = "test"
	DeviceClassCount = 5
)

type ConfigService struct {
	pb.UnimplementedConfigServer
	uc     *biz.ConfigUsecase
	logger *log.Helper
}

func NewConfigService(uc *biz.ConfigUsecase, logger log.Logger) *ConfigService {
	return &ConfigService{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
}
func (s *ConfigService) SaveInitDeviceConfig(conn pb.Config_SaveInitDeviceConfigServer) error {
	for {
		req, err := conn.Recv()
		if err == io.EOF {
			return conn.SendAndClose(&pb.ConfigServiceReply{})
		}
		if err != nil {
			return err
		}
	}
}
func (s *ConfigService) CreateConfigUpdateStream(conn pb.Config_CreateConfigUpdateStreamServer) error {
	for {
		req, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = conn.Send(&pb.DeviceConfig{})
		if err != nil {
			return err
		}
	}
}
func (s *ConfigService) UpdateDeviceConfig(ctx context.Context, req *pb.DeviceConfig) (*pb.ConfigServiceReply, error) {
	return &pb.ConfigServiceReply{}, nil
}
