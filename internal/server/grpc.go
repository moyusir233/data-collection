package server

import (
	v1 "gitee.com/moyusir/data-collection/api/dataCollection/v1"
	"gitee.com/moyusir/data-collection/internal/conf"
	"gitee.com/moyusir/data-collection/internal/service"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	g "google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// NewGRPCServer new a gRPC server.
func NewGRPCServer(c *conf.Server, cs *service.ConfigService, ws *service.WarningDetectService, logger log.Logger) *grpc.Server {
	var opts = []grpc.ServerOption{
		grpc.Middleware(
			recovery.Recovery(
				recovery.WithLogger(logger),
			),
			logging.Server(logger),
		),
		grpc.Options(g.KeepaliveParams(keepalive.ServerParameters{
			Time:    c.Grpc.KeepAliveTime.AsDuration(),
			Timeout: c.Grpc.KeepAliveTimeout.AsDuration(),
		})),
	}
	if c.Grpc.Network != "" {
		opts = append(opts, grpc.Network(c.Grpc.Network))
	}
	if c.Grpc.Addr != "" {
		opts = append(opts, grpc.Address(c.Grpc.Addr))
	}
	if c.Grpc.Timeout != nil {
		opts = append(opts, grpc.Timeout(c.Grpc.Timeout.AsDuration()))
	}
	srv := grpc.NewServer(opts...)
	v1.RegisterConfigServer(srv, cs)
	v1.RegisterWarningDetectServer(srv, ws)
	return srv
}
