package service

import (
	"context"
	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
	"gitee.com/moyusir/dataCollection/internal/biz"
	util "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/log"
	"io"
)

type ConfigService struct {
	pb.UnimplementedConfigServer
	uc      *biz.ConfigUsecase
	manager *biz.RouteManager
	logger  *log.Helper
}

func NewConfigService(uc *biz.ConfigUsecase, r *biz.RouteManager, logger log.Logger) *ConfigService {
	return &ConfigService{
		uc:      uc,
		manager: r,
		logger:  log.NewHelper(logger),
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

		err = conn.Send(&util.TestedDeviceConfig{})
		if err != nil {
			return err
		}
	}
}
func (s *ConfigService) UpdateDeviceConfig(ctx context.Context, req *util.TestedDeviceConfig) (*pb.ConfigServiceReply, error) {
	return &pb.ConfigServiceReply{}, nil
}
