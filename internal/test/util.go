package test

import (
	"context"
	v1 "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/conf"
	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	g "google.golang.org/grpc"
	"os"
	"testing"
)

func newApp(logger log.Logger, hs *http.Server, gs *grpc.Server) *kratos.App {
	var (
		// Name is the name of the compiled software.
		Name string = "data-collection"
		// Version is the version of the compiled software.
		Version string = "v0.0.1"
		id, _          = os.Hostname()
	)

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

// 初始化测试环境的通用辅助函数,path为配置文件的导入路径，envs为需要设置的环境变量
func generalInit(path string, envs map[string]string) (*conf.Bootstrap, error) {
	// 初始化测试所需环境变量
	// 环境变量的默认选项
	defaultEnvs := map[string]string{
		"USERNAME": "test",
	}

	for k, v := range envs {
		defaultEnvs[k] = v
	}

	for k, v := range defaultEnvs {
		err := os.Setenv(k, v)
		if err != nil {
			return nil, err
		}
	}

	// 导入配置文件
	if path == "" {
		path = "../../configs/config.yaml"
	}
	bc, err := conf.LoadConfig(path, log.DefaultLogger)
	if err != nil {
		return nil, err
	}

	return bc, nil
}

// StartDataCollectionTestServer 开启提供dataCollection服务的测试服务器，并返回相应服务的客户端
func StartDataCollectionTestServer(t *testing.T, bootstrap *conf.Bootstrap) (
	v1.ConfigClient, v1.ConfigHTTPClient, v1.WarningDetectClient) {
	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
	)
	app, cleanUp, err := initApp(bootstrap.Server, bootstrap.Data, logger)
	if err != nil {
		t.Fatal(err)
	}

	// done用来等待服务器关闭完毕
	done := make(chan struct{})
	t.Cleanup(func() {
		app.Stop()
		cleanUp()
		<-done
	})
	go func() {
		defer close(done)
		err = app.Run()
		if err != nil {
			t.Error(err)
		}
	}()

	// 等待服务器开启，然后创建测试用的grpc、http与网关客户端以及使用的api密钥
	var (
		grpcConn            *g.ClientConn
		httpConn            *http.Client
		configClient        v1.ConfigClient
		warningDetectClient v1.WarningDetectClient
		configHttpClient    v1.ConfigHTTPClient
	)

	// 创建http与grpc的连接与客户端
initClient:
	for {
		select {
		case <-done:
			t.Fatal("server start fail")
		default:
			// 创建grpc客户端
			// 因为grpc服务的网关组件由服务中心创建，所以这里没有通过网关建立grpc连接，而是本地直接连接
			grpcConn, err = grpc.DialInsecure(
				context.Background(),
				grpc.WithEndpoint("localhost:9000"),
			)
			if err != nil {
				grpcConn.Close()
				continue
			} else {
				configClient = v1.NewConfigClient(grpcConn)
				warningDetectClient = v1.NewWarningDetectClient(grpcConn)
			}

			// 创建http客户端
			httpConn, err = http.NewClient(
				context.Background(),
				http.WithEndpoint("localhost:8000"),
			)

			if err != nil {
				httpConn.Close()
				continue
			} else {
				configHttpClient = v1.NewConfigHTTPClient(httpConn)
			}
			break initClient
		}
	}
	t.Cleanup(func() {
		grpcConn.Close()
		httpConn.Close()
	})

	return configClient, configHttpClient, warningDetectClient
}
