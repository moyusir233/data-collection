package test

import (
	"gitee.com/moyusir/data-collection/internal/data"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"testing"
)

func TestData_NewData(t *testing.T) {
	// 初始化测试所需环境变量
	bc, err := generalInit("", map[string]string{})
	if err != nil {
		t.Fatal(err)
	}

	_, cleanUp, err := data.NewData(bc.Data, log.NewStdLogger(os.Stdout))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cleanUp)
}
