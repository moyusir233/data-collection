package test

import (
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestData_NewData(t *testing.T) {
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
