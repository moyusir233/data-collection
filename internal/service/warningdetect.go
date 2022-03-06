package service

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/biz"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"io"
	"time"

	pb "gitee.com/moyusir/dataCollection/api/dataCollection/v1"
)

type WarningDetectService struct {
	pb.UnimplementedWarningDetectServer
	uc      *biz.WarningDetectUsecase
	manager *biz.RouteManager
	logger  *log.Helper
}

func NewWarningDetectService(uc *biz.WarningDetectUsecase, r *biz.RouteManager, logger log.Logger) *WarningDetectService {
	return &WarningDetectService{
		uc:      uc,
		manager: r,
		logger:  log.NewHelper(logger),
	}
}

// CreateStateInfoSaveStream 创建传输设备状态信息的数据流
// 程序会优先使用请求头中包含的clientID进行路由信息的创建，
// 当请求头中不包含clientID时，则会重新建立一个clientID。
// 本次数据流中使用的clientID会通过响应头发送给客户端
func (s *WarningDetectService) CreateStateInfoSaveStream(conn pb.WarningDetect_CreateStateInfoSaveStreamServer) error {
	var (
		clientID string
		// 设备类别号，代码生成时注入
		deviceClassID = 0
		info          = &biz.DeviceGeneralInfo{DeviceClassID: deviceClassID}
		// 设备预警字段，代码生成时注入
		fields = map[string]float64{
			"Voltage": 0,
			"Current": 0,
		}
	)

	// 检查请求头中是否包含clientID
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
		info.DeviceID = state.Id
		// TODO 这里不应该使用反射去提取值，而是通过代码生成提取
		fields["Voltage"] = state.Voltage
		fields["Current"] = state.Current

		// TODO 考虑路由激活以及保存设备状态出错时如何处理
		err = s.manager.ActivateRoute(clientID, info)
		if err != nil {
			return err
		}
		err = s.uc.SaveDeviceState(info, state, fields)
		if err != nil {
			return err
		}

		err = conn.Send(&pb.WarningDetectServiceReply{Success: true})
		if err != nil {
			return err
		}
	}
}
