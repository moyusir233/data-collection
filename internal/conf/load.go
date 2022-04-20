package conf

import (
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"os"
)

// 定义了当前程序服务的用户及其设备的基本信息，由服务中心生成代码时注入
var (
	// Username 该服务对应用户的id
	Username = "test"
)

func initEnv(logger log.Logger) {
	if username, ok := os.LookupEnv("USERNAME"); ok {
		Username = username
	} else {
		logger.Log(log.LevelFatal, "msg", "The required environment variable USERNAME is missing")
		os.Exit(1)
	}
}
func LoadConfig(path string, logger log.Logger) (*Bootstrap, error) {
	initEnv(logger)

	c := config.New(
		config.WithSource(
			file.NewSource(path),
		),
		config.WithLogger(logger),
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
