package biz

import (
	"context"
	"fmt"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/golang/protobuf/proto"
)

// DeviceUpdateChannelsKey 保存设备和其更新channel对应关系的hash的key
var DeviceUpdateChannelsKey = fmt.Sprintf("%s:device:update:channel", conf.Username)

// DeviceConfigUpdater 负责接收和发送配置更新的消息
type DeviceConfigUpdater struct {
	pubSubClient UnionRepo
	logger       *log.Helper
}

// PubSubClient 发布订阅的客户端
type PubSubClient interface {
	// GetMsgChannel 返回订阅频道
	GetMsgChannel(ctx context.Context, name string) (msgChan <-chan string, err error)
	// PublishMsg 向指定的channel发送消息
	PublishMsg(channel string, message ...string) error
	AddFieldValuePair(key, field, value string) error
	GetValueOfField(key, field string) (value string, err error)
	// CreateClientID 产生一个分布式全局唯一的clientID
	CreateClientID() (string, error)
}

func NewDeviceConfigUpdater(repo UnionRepo, logger log.Logger) *DeviceConfigUpdater {
	return &DeviceConfigUpdater{
		pubSubClient: repo,
		logger:       log.NewHelper(logger),
	}
}

// UpdateDeviceConfig 更新设备的配置
func (updater *DeviceConfigUpdater) UpdateDeviceConfig(info *DeviceGeneralInfo, config proto.Message) error {
	deviceKey := GetDeviceKey(info)
	updateChanName, err := updater.pubSubClient.GetValueOfField(DeviceUpdateChannelsKey, deviceKey)
	if err != nil {
		return err
	}

	// 将proto msg转换为十六进制字符串进行publish
	marshal, err := proto.Marshal(config)
	if err != nil {
		return errors.Newf(
			500, "Biz_Config_Error",
			"对设备配置信息进行protobuf序列化时发生了错误:%v", err,
		)
	}
	msg := fmt.Sprintf("%x", marshal)

	// 发布消息
	err = updater.pubSubClient.PublishMsg(updateChanName, msg)
	if err != nil {
		return err
	}

	return nil
}

// GetDeviceUpdateMsgChannel 获得推送clientID相关的配置更新消息的channel
// 这里直接将clientID作为了channel的名称
func (updater *DeviceConfigUpdater) GetDeviceUpdateMsgChannel(
	ctx context.Context, clientID string, protoTemplate proto.Message) (<-chan proto.Message, error) {
	msgChannel, err := updater.pubSubClient.GetMsgChannel(ctx, clientID)
	if err != nil {
		return nil, err
	}

	updateMsg := make(chan proto.Message)
	go func() {
		defer close(updateMsg)
		for {
			select {
			case <-ctx.Done():
			case msg := <-msgChannel:
				// 将十六进制的字符串转换为二进制信息，并反序列化为proto message
				// TODO 忽略反序列化失败的消息?
				var b []byte
				_, err := fmt.Sscanf(msg, "%x", &b)
				if err != nil {
					updater.logger.Errorf(
						"反序列化接收到的配置更新十六进制字符串时发生了错误:%v", err,
					)
					continue
				}

				err = proto.Unmarshal(b, protoTemplate)
				if err != nil {
					updater.logger.Errorf(
						"将设备配置的二进制信息反序列化为proto message时发生了错误:%v", err,
					)
					continue
				}
				updateMsg <- proto.Clone(protoTemplate)
			}
		}
	}()

	return updateMsg, nil
}

// ConnectDeviceAndClientID 建立clientID和设备的关联关系，
// 可以利用与设备相关联的clientID的channel接收相应设备的配置更新消息
func (updater *DeviceConfigUpdater) ConnectDeviceAndClientID(clientID string, info *DeviceGeneralInfo) error {
	// 由于clientID即为clientID相应的channel的名称，因此直接
	// 建立添加设备key和clientID的键值对即可
	err := updater.pubSubClient.AddFieldValuePair(
		DeviceUpdateChannelsKey, GetDeviceKey(info), clientID)
	if err != nil {
		return err
	}

	return nil
}

// CreateClientID 产生一个分布式全局唯一的clientID
func (updater *DeviceConfigUpdater) CreateClientID() (string, error) {
	return updater.pubSubClient.CreateClientID()
}

// GetDeviceKey 辅助函数
func GetDeviceKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
}
