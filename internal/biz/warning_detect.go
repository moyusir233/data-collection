package biz

import (
	"github.com/go-kratos/kratos/v2/log"
	"strconv"
	"time"
)

// WarningDetectFieldLabelName 在创建设备预警字段相应ts的标签时使用的标签名
const WarningDetectFieldLabelName = "field_id"

type WarningDetectUsecase struct {
	repo   WarningDetectRepo
	logger *log.Helper
}

type WarningDetectRepo interface {
	// SaveDeviceState 保存设备完整状态信息以及预警字段信息
	SaveDeviceState(measurement *DeviceStateMeasurement) error
}

// DeviceStateMeasurement 每个设备状态信息以measurement的形式保存到influxdb中
type DeviceStateMeasurement struct {
	Name   string
	Time   time.Time
	Tags   map[string]string
	Fields map[string]float64
}

func NewWarningDetectUsecase(repo UnionRepo, logger log.Logger) *WarningDetectUsecase {
	return &WarningDetectUsecase{
		repo:   repo,
		logger: log.NewHelper(logger),
	}
}

// SaveDeviceState 保存设备状态的完整信息以及预警字段信息,其中预警字段以<字段名>:<字段值>的map形式传入函数，
// 非时间字段的设备字段被视作tag，也以map形式传入
func (u *WarningDetectUsecase) SaveDeviceState(
	info *DeviceGeneralInfo,
	fields map[string]float64,
	tags map[string]string) error {
	// 设备的预警字段信息以influxdb measurement的形式，保存到用户id相应的bucket以及设备id相应的measurement
	// 中，并以tag deviceClassID区分设备类别，各个字段的信息以field的形式保存在measurement的field中，
	// 非时间的预警字段则作为measurement的tag保存进influxdb
	tags["deviceClassID"] = strconv.Itoa(info.DeviceClassID)
	measurement := &DeviceStateMeasurement{
		Name:   info.DeviceID,
		Tags:   tags,
		Fields: fields,
	}

	return u.repo.SaveDeviceState(measurement)
}
