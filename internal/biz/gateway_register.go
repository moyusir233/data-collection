package biz

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/util/kong"
	"github.com/go-kratos/kratos/v2/log"
	"golang.org/x/sync/errgroup"
	"sync"
	"time"
)

var (
	defaultServiceCreateOption    *kong.ServiceCreateOption
	defaultRouteCreateOption      *kong.RouteCreateOption
	defaultAuthPluginCreateOption *kong.KeyAuthPluginCreateOption
)

// GatewayRegister 负责路由的注册以及自动注销
type GatewayRegister struct {
	// 网关客户端
	gateway *kong.Admin
	// 路由的自动注销时间
	timeout time.Duration
	// route注册表
	table *sync.Map
	// 除了route以外，所有注册的网关组件，保存用于容器暂停服务时向网关注销组件
	objects []kong.Object
	// 用于多协程控制
	eg *errgroup.Group
}

func NewGatewayRegister(c *conf.Server, logger log.Logger) *GatewayRegister {
	// 初始化kong组件创建的默认配置
	defaultServiceCreateOption = &kong.ServiceCreateOption{
		Name:     conf.ServiceName,
		Protocol: "http",
		Host:     conf.ServiceHost,
		Port:     int(c.Http.Port),
		Path:     "/",
		Enabled:  true,
		Tags:     []string{conf.Username},
	}
	// route的name和headers在注册路由时动态填写
	defaultRouteCreateOption = &kong.RouteCreateOption{
		Name:      "",
		Protocols: []string{"http"},
		Methods:   []string{"POST", "GET"},
		Hosts:     []string{conf.AppDomainName},
		Headers: map[string][]string{
			"X-Device-ID": {""},
		},
		StripPath: false,
		Service: &struct {
			Name string `json:"name,omitempty"`
			Id   string `json:"id,omitempty"`
		}{Name: conf.ServiceName},
		Tags: []string{conf.Username},
	}
	// 注册与service相关联的用户认证插件
	defaultAuthPluginCreateOption = &kong.KeyAuthPluginCreateOption{
		Enabled: true,
		Service: &struct {
			Name string `json:"name,omitempty"`
			Id   string `json:"id,omitempty"`
		}{Name: conf.ServiceName},
		// 配置通过请求头进行认证
		Config: &kong.KeyAuthPluginConfig{
			KeyNames:    []string{"X-Api-Key"},
			KeyInQuery:  false,
			KeyInBody:   false,
			KeyInHeader: true,
		},
		Tags: []string{conf.Username},
	}

	return &GatewayRegister{
		gateway: kong.NewAdmin(c.Gateway.Address, logger),
		timeout: c.Gateway.RouteTimeout.AsDuration(),
		table:   new(sync.Map),
		eg:      new(errgroup.Group),
	}
}

// Init 创建服务的网关组件service以及plugin,route动态创建
func (r *GatewayRegister) Init() error {
	options := []interface{}{
		defaultServiceCreateOption,
		defaultAuthPluginCreateOption,
	}
	for _, o := range options {
		object, err := r.gateway.Create(o)
		if err != nil {
			return err
		}
		r.objects = append(r.objects, object)
		// 组件创建需要间隔一段时间
		time.Sleep(time.Second)
	}
	return nil
}

// Close 清理服务注册的相关网关组件
func (r *GatewayRegister) Close() error {
	var err error = nil
	// 依据创建组件时的倒序注销组件，避免由于组件依赖关系造成的无法删除错误
	for i := len(r.objects) - 1; i >= 0; i-- {
		err = r.gateway.Delete(r.objects[i])
	}
	// 注销所有路由
	r.table.Range(func(key, value interface{}) bool {
		value.(*time.Ticker).Reset(time.Nanosecond)
		return true
	})
	// 等待所有路由都被注销
	r.eg.Wait()
	return err
}

// ActivateRoute 激活给定设备的路由,对于已注册的路由重置计时器,对于未注册的路由则注册
func (r *GatewayRegister) ActivateRoute(info *DeviceGeneralInfo) error {
	key := fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
	if ticker, ok := r.table.Load(key); ok {
		// 通过重新设置定时器激活路由
		ticker.(*time.Ticker).Reset(r.timeout)
		return nil
	} else {
		// 创建设备配置更新使用的路由,以请求头和host作为路由匹配规则
		// 先写入路由表，避免多协程对单个路由多次注册
		ticker := time.NewTicker(r.timeout)
		r.table.Store(key, ticker)
		defaultRouteCreateOption.Name = key
		defaultRouteCreateOption.Headers["X-Device-ID"] = []string{key}
		route, err := r.gateway.Create(defaultRouteCreateOption)
		if err != nil {
			return err
		}
		r.eg.Go(func() error {
			r.autoUnRegister(ticker, route)
			return nil
		})
		return nil
	}
}

// UnRegisterRoute 注销路由
func (r *GatewayRegister) UnRegisterRoute(info *DeviceGeneralInfo) {
	key := fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
	if ticker, ok := r.table.Load(key); ok {
		// 通过设置定时器为1纳秒，快速触发路由的自动注销
		ticker.(*time.Ticker).Reset(time.Nanosecond)
	}
}
func (r *GatewayRegister) autoUnRegister(ticker *time.Ticker, route kong.Object) {
	<-ticker.C
	// 注销路由并删除路由表中信息
	// 先向网关注销路由，再删除路由表中信息，
	// 确保先注销路由，避免向网关注册路由和注销路由同时发生
	r.gateway.Delete(route)
	// 删除路由表信息后再关闭计时器,避免其他协程对已经关闭的计时器执行reset
	r.table.Delete(route.(*kong.Route).Name)
	ticker.Stop()
}
