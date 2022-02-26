package conf

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"os"
	"strconv"
)

// 定义了当前程序服务的用户及其设备的基本信息，由服务中心生成代码时注入
var (
	Username         = "test"
	DeviceClassCount = 5
	ServiceName      = "test"
)

func init() {
	if username, ok := os.LookupEnv("USERNAME"); ok {
		Username = username
	} else {
		Username = "test"
	}
	if count, ok := os.LookupEnv("DEVICE_CLASS_COUNT"); ok {
		DeviceClassCount, _ = strconv.Atoi(count)
	} else {
		DeviceClassCount = 5
	}
	if serviceName, ok := os.LookupEnv("SERVICE_NAME"); ok {
		ServiceName = serviceName
	} else {
		serviceName = "test"
	}
}
func LoadConfig(path string) (*Bootstrap, error) {
	c := config.New(
		config.WithSource(
			file.NewSource(path),
		),
	)
	defer c.Close()

	if err := c.Load(); err != nil {
		return nil, err
	}

	var bc Bootstrap
	if err := c.Scan(&bc); err != nil {
		return nil, err
	}
	return &bc, nil
}
