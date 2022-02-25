package service

import (
	"gitee.com/moyusir/dataCollection/internal/biz"
	"github.com/go-kratos/kratos/v2/log"
	"io"

	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
)

type WarningDetectService struct {
	pb.UnimplementedWarningDetectServer
	uc     *biz.WarningDetectUsecase
	logger *log.Helper
}

func NewWarningDetectService(uc *biz.WarningDetectUsecase, logger log.Logger) *WarningDetectService {
	return &WarningDetectService{
		uc:     uc,
		logger: log.NewHelper(logger),
	}
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
