package biz

import (
	"fmt"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/google/wire"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewConfigUsecase, NewWarningDetectUsecase, NewDeviceConfigUpdater)

// DeviceGeneralInfo 设备基本信息
type DeviceGeneralInfo struct {
	DeviceClassID int
	DeviceID      string
}

// UnionRepo 为了方便wire注入使用，这里合并两个repo接口
type UnionRepo interface {
	ConfigRepo
	WarningDetectRepo
	PubSubClient
}

// StateProtoMessage 用于规范设备状态信息定义的接口
type StateProtoMessage interface {
	proto.Message
	GetTime() *timestamppb.Timestamp
}

// GetDeviceConfigKey 以<用户id>:device_config:<device_class_id>:hash为键
// ,以设备id为field,在redis hash中保存设备配置的protobuf二进制信息
func GetDeviceConfigKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s:device_config:%d:hash", conf.Username, info.DeviceClassID)
}

// GetDeviceStateKey 以<用户id>:device_state:<设备类别号>为键，在zset中保存
// 以timestamp为score，以设备状态二进制protobuf信息为value的键值对
func GetDeviceStateKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s:device_state:%d", conf.Username, info.DeviceClassID)
}

// GetDeviceStateFieldKeyAndLabel 每个预警字段保存到以<用户id>:device_state:<设备类别号>:<设备字段名>:<设备id>为key，
// 以field_id为标签名，以<用户id>:<设备类别号>:<字段名>为标签值的ts中
func GetDeviceStateFieldKeyAndLabel(info *DeviceGeneralInfo, fieldName string) (key, label string) {
	key = fmt.Sprintf(
		"%s:device_state:%d:%s:%s",
		conf.Username,
		info.DeviceClassID,
		fieldName,
		info.DeviceID,
	)
	label = fmt.Sprintf("%s:%d:%s",
		conf.Username,
		info.DeviceClassID,
		fieldName,
	)
	return
}
