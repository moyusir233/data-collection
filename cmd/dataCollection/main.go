package main

import (
	"flag"
	"fmt"
	"os"

	"gitee.com/moyusir/data-collection/internal/conf"
	util "gitee.com/moyusir/util/logger"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string = "data-collection"
	// Version is the version of the compiled software.
	Version string = "v0.0.1"
	// flagconf is the config flag.
	flagconf string

	id, _ = os.Hostname()
)

func init() {
	flag.StringVar(&flagconf, "conf", "../../configs", "config path, eg: -conf config.yaml")
}

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			hs,
			gs,
		),
	)
}

func main() {
	flag.Parse()

	bc, err := conf.LoadConfig(flagconf, log.DefaultLogger)
	if err != nil {
		panic(fmt.Sprintf("导入配置时发生了错误:%v", err))
	}

	logger := util.NewJsonZapLoggerWarpper(Name, bc.LogLevel)
	helper := log.NewHelper(logger)

	app, cleanup, err := initApp(bc.Server, bc.Data, logger)
	if err != nil {
		helper.Fatalf("应用初始化时发生了错误:%v", err)
	}
	defer cleanup()

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		helper.Fatalf("应用运行时发生了错误:%v", err)
	}
}
