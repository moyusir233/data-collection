package biz

type ConfigUsecase struct {
}
type ConfigRepo interface {
	// SaveDeviceConfig 保存设备配置信息
	SaveDeviceConfig(key, field string, value []byte) error
}

func NewConfigUsecase() *ConfigUsecase {
	return nil
}
