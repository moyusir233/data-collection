package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
)

// WarningDetectFieldLabelName 在创建设备预警字段相应ts的标签时使用的标签名
const WarningDetectFieldLabelName = "field_id"

type WarningDetectUsecase struct {
	repo   WarningDetectRepo
	logger *log.Helper
}

type WarningDetectRepo interface {
	// SaveDeviceState 保存设备完整状态信息以及预警字段信息
	SaveDeviceState(state *DeviceState, fields ...*DeviceStateField) error
}

// DeviceState 代表一台设备完整的状态信息，利用protobuf编码，编码为二进制信息保存
type DeviceState struct {
	Timestamp int64
	Key       string
	Value     []byte
}

// DeviceStateField 代表设备状态信息中需要进行预警预测的字段
type DeviceStateField struct {
	Key   string
	Value float64
	Label string
}

func NewWarningDetectUsecase(repo UnionRepo, logger log.Logger) *WarningDetectUsecase {
	return &WarningDetectUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// SaveDeviceState 保存设备状态的完整信息以及预警字段信息,其中预警字段以<字段名>:<字段值>的形式传入函数
func (u *WarningDetectUsecase) SaveDeviceState(info *DeviceGeneralInfo, state StateProtoMessage, fields map[string]float64) error {
	var (
		s = new(DeviceState)
		f = make([]*DeviceStateField, 0, len(fields))
	)
	// 以<用户id>:device_state:<设备类别号>为键，在zset中保存
	// 以timestamp为score，以设备状态二进制protobuf信息为value的键值对
	s.Key = GetDeviceStateKey(info)
	s.Timestamp = state.GetTime().AsTime().UnixMilli()
	marshal, err := proto.Marshal(state)
	if err != nil {
		return err
	}
	s.Value = marshal

	for k, v := range fields {
		field := new(DeviceStateField)
		// 每个预警字段保存到以<用户id>:device_state:<设备类别号>:<设备字段名>:<设备id>为key，
		// 以<用户id>:<设备类别号>:<字段名>为标签值的ts中
		field.Key, field.Label = GetDeviceStateFieldKeyAndLabel(info, k)
		field.Value = v
		f = append(f, field)
	}
	err = u.repo.SaveDeviceState(s, f...)
	if err != nil {
		return err
	}
	return nil
}
