package data

import (
	"gitee.com/moyusir/dataCollection/internal/conf"
	"github.com/go-kratos/kratos/v2/config"
	"github.com/go-kratos/kratos/v2/config/file"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestData_NewData(t *testing.T) {
	c := config.New(config.WithSource(file.NewSource("../../configs/config.yaml")))
	if err := c.Load(); err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	var bc conf.Bootstrap
	if err := c.Scan(&bc); err != nil {
		t.Fatal(err)
	}
	_, cleanUp, err := NewData(bc.Data, log.NewStdLogger(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	defer cleanUp()
	t.Logf("%s测试成功", t.Name())
}
