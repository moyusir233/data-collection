package data

import (
	"context"
	"fmt"
	"gitee.com/moyusir/data-collection/internal/biz"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/influxdata/influxdb-client-go/v2/api/write"
	"time"
)

// Repo redis数据库操作对象，可以理解为dao
type Repo struct {
	redisClient    *RedisData
	influxdbClient *InfluxdbData
	logger         *log.Helper
}

// NewRepo 实例化redis数据库操作对象
func NewRepo(redisData *RedisData, influxdbData *InfluxdbData, logger log.Logger) biz.UnionRepo {
	return &Repo{
		redisClient:    redisData,
		influxdbClient: influxdbData,
		logger:         log.NewHelper(logger),
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
	// 非时间、非id且非预警的字段则作为measurement的tag保存进influxdb
	writeAPI := r.influxdbClient.WriteAPIBlocking(r.influxdbClient.org, conf.Username)

	point := write.NewPointWithMeasurement(measurement.Name).SetTime(measurement.Time.UTC())
	for k, v := range measurement.Tags {
		point.AddTag(k, v)
	}
	for k, v := range measurement.Fields {
		point.AddField(k, v)
	}
	point.SortFields().SortTags()

	err := writeAPI.WritePoint(context.Background(), point)
	if err != nil {
		return errors.Newf(
			500, "Repo_State_Error", "设备状态保存时发生了错误:%v", err)
	}

	{
		now := time.Now().UTC()
		r.logger.Debugf(
			"与时间:%s保存了时间信息为:%s的设备状态信息,时间差为:%s",
			now.Format(time.RFC3339), measurement.Time.UTC().Format(time.RFC3339),
			now.Sub(measurement.Time.UTC()).String(),
		)
	}

	return nil
}

func (r *Repo) GetMsgChannel(ctx context.Context, name string) (msgChan <-chan string, err error) {
	subscribe := r.redisClient.Subscribe(ctx, name)
	r.logger.Infof("订阅了频道:%v", name)
	messages := subscribe.Channel()
	messageChan := make(chan string, 100)
	go func() {
		defer func() {
			close(messageChan)
			err := subscribe.Close()
			if err != nil {
				r.logger.Errorf("关闭 %v 的订阅对象时发生了错误:%v", name, err)
				return
			}
			r.logger.Infof("取消了 %v 频道的订阅", name)
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-messages:
				if msg == nil {
					continue
				}
				if msg.Payload != "" {
					messageChan <- msg.Payload
				}
				for _, m := range msg.PayloadSlice {
					if m != "" {
						messageChan <- m
					}
				}
			}
		}
	}()

	return messageChan, nil
}

func (r *Repo) PublishMsg(channel string, message ...string) error {
	for _, msg := range message {
		err := r.redisClient.Publish(context.Background(), channel, msg).Err()
		if err != nil {
			return errors.Newf(
				500, "Repo_Config_Error", "发布消息时发生了错误:%v", err)
		}
	}
	return nil
}

func (r *Repo) AddFieldValuePair(key, field, value string) error {
	err := r.redisClient.HSet(context.Background(), key, field, value).Err()
	if err != nil {
		return errors.Newf(
			500, "Repo_Config_Error", "保存hash键值对时发生了错误:%v", err)
	}

	return nil
}

func (r *Repo) GetValueOfField(key, field string) (value string, err error) {
	value, err = r.redisClient.HGet(context.Background(), key, field).Result()
	if err != nil {
		return "", errors.Newf(
			500, "Repo_Config_Error", "查询hash键值对时发生了错误:%v", err)
	}

	return
}

// CreateClientID 利用redis的自增函数产生分布式全局唯一的clientID
func (r *Repo) CreateClientID() (string, error) {
	result, err := r.redisClient.HIncrBy(
		context.Background(), "clientID", conf.Username, 1).Result()
	if err != nil {
		return "", errors.Newf(
			500, "Repo_Config_Error", "创建clientID时发生了错误:%v", err)
	}

	return fmt.Sprintf("%s_%d", conf.Username, result), nil
}
