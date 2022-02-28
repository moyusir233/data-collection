package biz

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/util/kong"
	"golang.org/x/sync/errgroup"
	"time"
)

var (
	defaultServiceCreateOption    *kong.ServiceCreateOption
	defaultRouteCreateOption      *kong.RouteCreateOption
	defaultAuthPluginCreateOption *kong.KeyAuthPluginCreateOption
)

// RouteManager 负责管理路由，包括api网关的路由以及服务内部关于设备更新连接的路由
// todo 修改路由自动注销的机制：将单个路由逐个注销改为利用tag批量注销相关联的所有route
// todo 修改Close函数，改为利用tag进行批量注销组件
type RouteManager struct {
	// 网关客户端
	gateway *kong.Admin
	// 路由的自动注销时间
	timeout time.Duration
	// route注册表
	table *RouteTable
	// 除了route以外，所有注册的网关组件，保存用于容器暂停服务时向网关注销组件
	objects []kong.Object
	// 用于多协程控制
	eg *errgroup.Group
}

func NewRouteManager(c *conf.Server) *RouteManager {
	// 初始化kong组件创建的默认配置
	// 设备配置更新的相关路由组件都打上了conf.ServiceName的tag(即pod的名字)
	// 方便容器关闭时组件的注销
	defaultServiceCreateOption = &kong.ServiceCreateOption{
		Name:     conf.ServiceName,
		Protocol: "http",
		Host:     conf.ServiceHost,
		Port:     int(c.Http.Port),
		Path:     "/",
		Enabled:  true,
		Tags:     []string{conf.Username, conf.ServiceName},
	}
	// route的name和headers在注册路由时动态填写
	defaultRouteCreateOption = &kong.RouteCreateOption{
		Name:      "",
		Protocols: []string{"http"},
		Methods:   []string{"POST", "GET"},
		Hosts:     []string{conf.AppDomainName},
		Paths:     []string{"/"},
		Headers: map[string][]string{
			"X-Device-ID": {""},
		},
		StripPath: false,
		Service: &struct {
			Name string `json:"name,omitempty"`
			Id   string `json:"id,omitempty"`
		}{Name: conf.ServiceName},
		Tags: []string{conf.Username, conf.ServiceName},
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
		Tags: []string{conf.Username, conf.ServiceName},
	}

	return &RouteManager{
		gateway: kong.NewAdmin(c.Gateway.Address),
		timeout: c.Gateway.RouteTimeout.AsDuration(),
		table:   new(RouteTable),
		eg:      new(errgroup.Group),
	}
}

// Init 创建服务的网关组件service以及plugin,route动态创建
func (r *RouteManager) Init() error {
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
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// Close 清理服务注册的相关网关组件
func (r *RouteManager) Close() error {
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
// todo 利用RouteTable重新实现路由激活的逻辑，多个相同key的激活请求与多个不同key的激活请求都能否保证协程安全？
func (r *RouteManager) ActivateRoute(info *DeviceGeneralInfo) error {
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
		option := kong.RouteCreateOption{
			Name:      key,
			Protocols: defaultRouteCreateOption.Protocols,
			Methods:   defaultRouteCreateOption.Methods,
			Hosts:     defaultRouteCreateOption.Hosts,
			Paths:     defaultRouteCreateOption.Paths,
			Headers: map[string][]string{
				"X-Device-ID": {key},
			},
			StripPath: defaultRouteCreateOption.StripPath,
			Service:   defaultRouteCreateOption.Service,
			Tags:      defaultRouteCreateOption.Tags,
		}
		route, err := r.gateway.Create(&option)
		// route创建失败大部分情况下是由于客户端重连，被负载均衡到了其他的服务容器上
		// 导致原来的route没有删除，此时更新route即可
		if err != nil {
			route = new(kong.Route)
			err = route.(*kong.Route).Update(&option)
		}
		r.eg.Go(func() error {
			r.autoUnRegister(ticker, route)
			return nil
		})
		return err
	}
}

// UnRegisterRoute 注销路由
func (r *RouteManager) UnRegisterRoute(info *DeviceGeneralInfo) {
	key := fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
	if ticker, ok := r.table.Load(key); ok {
		// 通过设置定时器为1纳秒，快速触发路由的自动注销
		ticker.(*time.Ticker).Reset(time.Nanosecond)
	}
}

// 自动注销仅仅是将节点对应的若干route信息注销，并将节点的RouteTag标志为空
// 而不会删除节点在路由表中存储的信息，只有当协程检测到客户端的连接正常断开时
// 才会进行节点的路由注销以及节点信息的删除操作，包括关闭其配置更新channel等
func (r *RouteManager) autoUnRegister(node *RouteTableNode) {
	<-node.UnregisterTicker.C
	// 注销路由并删除路由表中信息
	// 先向网关注销路由，再删除路由表中信息，
	// 确保先注销路由，避免向网关注册路由和注销路由同时发生
	r.gateway.Delete(route)
	// 删除路由表信息后再关闭计时器,避免其他协程对已经关闭的计时器执行reset
	r.table.Delete(route.(*kong.Route).Name)
	ticker.Stop()
}
