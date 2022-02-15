// Code generated by protoc-gen-go-grpc. DO NOT EDIT.
// versions:
// - protoc-gen-go-grpc v1.2.0
// - protoc             v3.17.3
// source: api/dataCollection/v1/config.proto

package v1

import (
	context "context"
	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the grpc package it is being compiled against.
// Requires gRPC-Go v1.32.0 or later.
const _ = grpc.SupportPackageIsVersion7

// ConfigClient is the client API for Config service.
//
// For semantics around ctx use and closing/ending streaming RPCs, please refer to https://pkg.go.dev/google.golang.org/grpc/?tab=doc#ClientConn.NewStream.
type ConfigClient interface {
	// 更新指定设备的配置
	UpdateDeviceConfig(ctx context.Context, in *DeviceConfig, opts ...grpc.CallOption) (*ConfigServiceReply, error)
	// 用于和底层设备客户端建立用于配置更新的grpc数据流
	CreateConfigUpdateStream(ctx context.Context, opts ...grpc.CallOption) (Config_CreateConfigUpdateStreamClient, error)
	// 从底层数据客户端收集设备初始配置
	SaveInitDeviceConfig(ctx context.Context, opts ...grpc.CallOption) (Config_SaveInitDeviceConfigClient, error)
}

type configClient struct {
	cc grpc.ClientConnInterface
}

func NewConfigClient(cc grpc.ClientConnInterface) ConfigClient {
	return &configClient{cc}
}

func (c *configClient) UpdateDeviceConfig(ctx context.Context, in *DeviceConfig, opts ...grpc.CallOption) (*ConfigServiceReply, error) {
	out := new(ConfigServiceReply)
	err := c.cc.Invoke(ctx, "/api.dataCollection.v1.Config/UpdateDeviceConfig", in, out, opts...)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func (c *configClient) CreateConfigUpdateStream(ctx context.Context, opts ...grpc.CallOption) (Config_CreateConfigUpdateStreamClient, error) {
	stream, err := c.cc.NewStream(ctx, &Config_ServiceDesc.Streams[0], "/api.dataCollection.v1.Config/CreateConfigUpdateStream", opts...)
	if err != nil {
		return nil, err
	}
	x := &configCreateConfigUpdateStreamClient{stream}
	return x, nil
}

type Config_CreateConfigUpdateStreamClient interface {
	Send(*ConfigServiceReply) error
	Recv() (*DeviceConfig, error)
	grpc.ClientStream
}

type configCreateConfigUpdateStreamClient struct {
	grpc.ClientStream
}

func (x *configCreateConfigUpdateStreamClient) Send(m *ConfigServiceReply) error {
	return x.ClientStream.SendMsg(m)
}

func (x *configCreateConfigUpdateStreamClient) Recv() (*DeviceConfig, error) {
	m := new(DeviceConfig)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func (c *configClient) SaveInitDeviceConfig(ctx context.Context, opts ...grpc.CallOption) (Config_SaveInitDeviceConfigClient, error) {
	stream, err := c.cc.NewStream(ctx, &Config_ServiceDesc.Streams[1], "/api.dataCollection.v1.Config/SaveInitDeviceConfig", opts...)
	if err != nil {
		return nil, err
	}
	x := &configSaveInitDeviceConfigClient{stream}
	return x, nil
}

type Config_SaveInitDeviceConfigClient interface {
	Send(*DeviceConfig) error
	CloseAndRecv() (*ConfigServiceReply, error)
	grpc.ClientStream
}

type configSaveInitDeviceConfigClient struct {
	grpc.ClientStream
}

func (x *configSaveInitDeviceConfigClient) Send(m *DeviceConfig) error {
	return x.ClientStream.SendMsg(m)
}

func (x *configSaveInitDeviceConfigClient) CloseAndRecv() (*ConfigServiceReply, error) {
	if err := x.ClientStream.CloseSend(); err != nil {
		return nil, err
	}
	m := new(ConfigServiceReply)
	if err := x.ClientStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// ConfigServer is the server API for Config service.
// All implementations must embed UnimplementedConfigServer
// for forward compatibility
type ConfigServer interface {
	// 更新指定设备的配置
	UpdateDeviceConfig(context.Context, *DeviceConfig) (*ConfigServiceReply, error)
	// 用于和底层设备客户端建立用于配置更新的grpc数据流
	CreateConfigUpdateStream(Config_CreateConfigUpdateStreamServer) error
	// 从底层数据客户端收集设备初始配置
	SaveInitDeviceConfig(Config_SaveInitDeviceConfigServer) error
	mustEmbedUnimplementedConfigServer()
}

// UnimplementedConfigServer must be embedded to have forward compatible implementations.
type UnimplementedConfigServer struct {
}

func (UnimplementedConfigServer) UpdateDeviceConfig(context.Context, *DeviceConfig) (*ConfigServiceReply, error) {
	return nil, status.Errorf(codes.Unimplemented, "method UpdateDeviceConfig not implemented")
}
func (UnimplementedConfigServer) CreateConfigUpdateStream(Config_CreateConfigUpdateStreamServer) error {
	return status.Errorf(codes.Unimplemented, "method CreateConfigUpdateStream not implemented")
}
func (UnimplementedConfigServer) SaveInitDeviceConfig(Config_SaveInitDeviceConfigServer) error {
	return status.Errorf(codes.Unimplemented, "method SaveInitDeviceConfig not implemented")
}
func (UnimplementedConfigServer) mustEmbedUnimplementedConfigServer() {}

// UnsafeConfigServer may be embedded to opt out of forward compatibility for this service.
// Use of this interface is not recommended, as added methods to ConfigServer will
// result in compilation errors.
type UnsafeConfigServer interface {
	mustEmbedUnimplementedConfigServer()
}

func RegisterConfigServer(s grpc.ServiceRegistrar, srv ConfigServer) {
	s.RegisterService(&Config_ServiceDesc, srv)
}

func _Config_UpdateDeviceConfig_Handler(srv interface{}, ctx context.Context, dec func(interface{}) error, interceptor grpc.UnaryServerInterceptor) (interface{}, error) {
	in := new(DeviceConfig)
	if err := dec(in); err != nil {
		return nil, err
	}
	if interceptor == nil {
		return srv.(ConfigServer).UpdateDeviceConfig(ctx, in)
	}
	info := &grpc.UnaryServerInfo{
		Server:     srv,
		FullMethod: "/api.dataCollection.v1.Config/UpdateDeviceConfig",
	}
	handler := func(ctx context.Context, req interface{}) (interface{}, error) {
		return srv.(ConfigServer).UpdateDeviceConfig(ctx, req.(*DeviceConfig))
	}
	return interceptor(ctx, in, info, handler)
}

func _Config_CreateConfigUpdateStream_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ConfigServer).CreateConfigUpdateStream(&configCreateConfigUpdateStreamServer{stream})
}

type Config_CreateConfigUpdateStreamServer interface {
	Send(*DeviceConfig) error
	Recv() (*ConfigServiceReply, error)
	grpc.ServerStream
}

type configCreateConfigUpdateStreamServer struct {
	grpc.ServerStream
}

func (x *configCreateConfigUpdateStreamServer) Send(m *DeviceConfig) error {
	return x.ServerStream.SendMsg(m)
}

func (x *configCreateConfigUpdateStreamServer) Recv() (*ConfigServiceReply, error) {
	m := new(ConfigServiceReply)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

func _Config_SaveInitDeviceConfig_Handler(srv interface{}, stream grpc.ServerStream) error {
	return srv.(ConfigServer).SaveInitDeviceConfig(&configSaveInitDeviceConfigServer{stream})
}

type Config_SaveInitDeviceConfigServer interface {
	SendAndClose(*ConfigServiceReply) error
	Recv() (*DeviceConfig, error)
	grpc.ServerStream
}

type configSaveInitDeviceConfigServer struct {
	grpc.ServerStream
}

func (x *configSaveInitDeviceConfigServer) SendAndClose(m *ConfigServiceReply) error {
	return x.ServerStream.SendMsg(m)
}

func (x *configSaveInitDeviceConfigServer) Recv() (*DeviceConfig, error) {
	m := new(DeviceConfig)
	if err := x.ServerStream.RecvMsg(m); err != nil {
		return nil, err
	}
	return m, nil
}

// Config_ServiceDesc is the grpc.ServiceDesc for Config service.
// It's only intended for direct use with grpc.RegisterService,
// and not to be introspected or modified (even as a copy)
var Config_ServiceDesc = grpc.ServiceDesc{
	ServiceName: "api.dataCollection.v1.Config",
	HandlerType: (*ConfigServer)(nil),
	Methods: []grpc.MethodDesc{
		{
			MethodName: "UpdateDeviceConfig",
			Handler:    _Config_UpdateDeviceConfig_Handler,
		},
	},
	Streams: []grpc.StreamDesc{
		{
			StreamName:    "CreateConfigUpdateStream",
			Handler:       _Config_CreateConfigUpdateStream_Handler,
			ServerStreams: true,
			ClientStreams: true,
		},
		{
			StreamName:    "SaveInitDeviceConfig",
			Handler:       _Config_SaveInitDeviceConfig_Handler,
			ClientStreams: true,
		},
	},
	Metadata: "api/dataCollection/v1/config.proto",
}
