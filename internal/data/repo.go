package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/data-collection/internal/biz"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
)

// Repo redis数据库操作对象，可以理解为dao
type Repo struct {
	redisClient    *RedisData
	influxdbClient *InfluxdbData
}

// NewRepo 实例化redis数据库操作对象
func NewRepo(redisData *RedisData, influxdbData *InfluxdbData) biz.UnionRepo {
	return &Repo{
		redisClient:    redisData,
		influxdbClient: influxdbData,
	}
}

// SaveDeviceConfig 保存设备配置到redis的hash中
func (r *Repo) SaveDeviceConfig(key, field string, value []byte) error {
	// 这里将value转换为十六进制的字符串进行保存
	v := fmt.Sprintf("%x", value)
	if err := r.redisClient.HSet(context.Background(), key, field, v).Err(); err != nil {
		return errors.Newf(
			500, "Repo_Config_Error", "设备配置保存时发生了错误:%v", err)
	}
	return nil
}

// SaveDeviceState 保存设备状态的measurement
func (r *Repo) SaveDeviceState(measurement *biz.DeviceStateMeasurement) error {

	// 设备的预警字段信息以influxdb measurement的形式，保存到用户id相应的bucket以及设备id相应的measurement
	// 中，并以tag deviceClassID区分设备类别，各个字段的信息以field的形式保存在measurement的field中，
	// 非时间的预警字段则作为measurement的tag保存进influxdb
	// TODO 使用异步写入的api时如何进行错误处理？
	writeAPI := r.influxdbClient.WriteAPI(r.influxdbClient.org, conf.Username)

	point := write.NewPointWithMeasurement(measurement.Name).SetTime(measurement.Time.UTC())
	for k, v := range measurement.Tags {
		point.AddTag(k, v)
	}
	for k, v := range measurement.Fields {
		point.AddField(k, v)
	}
	point.SortFields().SortTags()

	writeAPI.WritePoint(point)
	return nil
}
