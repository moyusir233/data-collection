package service

import (
	"io"

	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
)

type WarningDetectService struct {
	pb.UnimplementedWarningDetectServer
}

func NewWarningDetectService() *WarningDetectService {
	return &WarningDetectService{}
}

func (s *WarningDetectService) SaveDeviceStateInfo(conn pb.WarningDetect_SaveDeviceStateInfoServer) error {
	for {
		req, err := conn.Recv()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}

		err = conn.Send(&pb.WarningDetectServiceReply{})
		if err != nil {
			return err
		}
	}
}
