package biz

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/service"
	"github.com/google/wire"
)

// ProviderSet is biz providers.
var ProviderSet = wire.NewSet(NewConfigUsecase, NewWarningDetectUsecase)

// DeviceGeneralInfo 设备基本信息
type DeviceGeneralInfo struct {
	DeviceClassID int
	DeviceID      string
}

// UnionRepo 为了方便wire注入使用，这里合并两个repo接口
type UnionRepo interface {
	ConfigRepo
	WarningDetectRepo
}

// GetDeviceConfigKey 以<用户id>:device_config:<device_class_id>:hash为键
// ,以设备id为field,在redis hash中保存设备配置的protobuf二进制信息
func GetDeviceConfigKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s:device_config:%d:hash", service.Username, info.DeviceClassID)
}

// GetDeviceStateKey 以<用户id>:device_state:<设备类别号>为键，在zset中保存
// 以timestamp为score，以设备状态二进制protobuf信息为value的键值对
func GetDeviceStateKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s:device_state:%d", service.Username, info.DeviceClassID)
}

// GetDeviceStateFieldKey 每个预警字段保存到以<用户id>:device_state:<设备类别号>:<设备字段名>:<设备id>的为key的ts中
func GetDeviceStateFieldKey(info *DeviceGeneralInfo, fieldName string) string {
	return fmt.Sprintf(
		"%s:device_state:%d:%s:%s",
		service.Username,
		info.DeviceClassID,
		fieldName,
		info.DeviceID,
	)
}
