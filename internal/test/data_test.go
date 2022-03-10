package test

import (
	"gitee.com/moyusir/data-collection/internal/conf"
	"gitee.com/moyusir/data-collection/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestData_NewData(t *testing.T) {
	// 初始化测试所需环境变量
	envs := map[string]string{
		"USERNAME":           "test",
		"DEVICE_CLASS_COUNT": "",
		"SERVICE_NAME":       "",
		"SERVICE_HOST":       "",
		"APP_DOMAIN_NAME":    "",
	}
	for k, v := range envs {
		err := os.Setenv(k, v)
		if err != nil {
			t.Fatal(err)
		}
	}

	bc, err := conf.LoadConfig("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	_, cleanUp, err := data.NewData(bc.Data, log.NewStdLogger(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanUp()
}
