package conf

import (
	"fmt"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"log"
	"os"
)

// 定义了当前程序服务的用户及其设备的基本信息，由服务中心生成代码时注入
var (
	// Username 该服务对应用户的id
	Username = "test"
	// ServiceName 注册kong service时使用的服务名,默认为pod的名称,通过环境变量注入
	ServiceName = "test"
	// ServiceHost 注册kong service时使用的host,为pod在k8s中注册的域名,通过环境变量注入
	ServiceHost = "test"
	// AppDomainName 应用注册的网站域名，用于注册kong route时的host匹配
	AppDomainName = "gd-k8s-master01"
)

func initEnv() {
	if username, ok := os.LookupEnv("USERNAME"); ok {
		Username = username
	} else {
		log.Fatalln("The required environment variable USERNAME is missing")
	}

	if serviceName, ok := os.LookupEnv("SERVICE_NAME"); ok {
		ServiceName = serviceName
	} else {
		log.Fatalln("The required environment variable SERVICE_NAME is missing")
	}

	if headlessServiceName, ok := os.LookupEnv("HEADLESS_SERVICE_NAME"); ok {
		ServiceHost = fmt.Sprintf(
			"%s.%s.test.svc.cluster.local", ServiceName, headlessServiceName)
	} else {
		log.Fatalln("The required environment variable SERVICE_HOST is missing")
	}

	if domainName, ok := os.LookupEnv("APP_DOMAIN_NAME"); ok {
		AppDomainName = domainName
	} else {
		log.Fatalln("The required environment variable APP_DOMAIN_NAME is missing")
	}
}
func LoadConfig(path string) (*Bootstrap, error) {
	initEnv()

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
