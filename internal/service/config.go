package service

import (
	"context"
	"io"

	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
)

type ConfigService struct {
	pb.UnimplementedConfigServer
}

func NewConfigService() *ConfigService {
	return &ConfigService{}
}

func (s *ConfigService) UpdateDeviceConfig(ctx context.Context, req *pb.DeviceConfig) (*pb.ConfigServiceReply, error) {
	return &pb.ConfigServiceReply{}, nil
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
