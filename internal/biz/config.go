package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"
)

type ConfigUsecase struct {
	repo   ConfigRepo
	logger *log.Helper
}
type ConfigRepo interface {
	// SaveDeviceConfig 保存设备配置信息
	SaveDeviceConfig(key, field string, value []byte) error
}

func NewConfigUsecase(repo UnionRepo, logger log.Logger) *ConfigUsecase {
	return &ConfigUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// SaveDeviceConfig 保存指定用户名以及设备类别号下的设备信息
func (u *ConfigUsecase) SaveDeviceConfig(info *DeviceGeneralInfo, config proto.Message) error {
	// 以<用户id>:device_config:<device_class_id>:hash为键
	// ,以设备id为field,在redis hash中保存设备配置的protobuf二进制信息
	key := getDeviceConfigKey(info)
	marshal, err := proto.Marshal(config)
	if err != nil {
		return err
	}
	err = u.repo.SaveDeviceConfig(key, info.DeviceID, marshal)
	if err != nil {
		return err
	}
	return nil
}
