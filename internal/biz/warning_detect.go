package biz

type WarningDetectUsecase struct {
}

type WarningDetectRepo interface {
	// SaveDeviceState 保存设备完整状态信息以及预警字段信息
	SaveDeviceState(state *DeviceState, fields ...*DeviceStateField) error
}

// DeviceState 代表一台设备完整的状态信息，利用protobuf编码，编码为二进制信息保存
type DeviceState struct {
	Key   string
	Value []byte
}

// DeviceStateField 代表设备状态信息中需要进行预警预测的字段
type DeviceStateField struct {
	Key   string
	Value float64
}

func NewWarningDetectUsecase() *WarningDetectUsecase {
	return nil
}
