package biz

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/dataCollection/internal/service"
	"gitee.com/moyusir/util/kong"
	"github.com/go-kratos/kratos/v2/log"
	"os"
	"sync"
	"time"
)

// RouteRegister 负责路由的注册以及自动注销
type RouteRegister struct {
	// 网关客户端
	gateway *kong.Admin
	// 路由的自动注销时间
	timeout time.Duration
	// 与路由关联注册的本机服务名
	serviceName string
	// 注册表
	table *sync.Map
}

func NewRouteRegister(c *conf.Server, logger log.Logger) *RouteRegister {
	name, _ := os.LookupEnv("SERVICE_NAME")
	return &RouteRegister{
		gateway:     kong.NewAdmin(c.Gateway.Address, logger),
		timeout:     c.Gateway.RouteTimeout.AsDuration(),
		serviceName: name,
		table:       new(sync.Map),
	}
}

// Activate 激活给定设备的路由,对于已注册的路由重置计时器,对于未注册的路由则注册
func (r *RouteRegister) Activate(info *DeviceGeneralInfo) error {
	key := fmt.Sprintf("%s_%d_%s", service.Username, info.DeviceClassID, info.DeviceID)
	if ticker, ok := r.table.Load(key); ok {
		// 通过重新设置定时器激活路由
		ticker.(*time.Ticker).Reset(r.timeout)
		return nil
	} else {
		// 创建设备配置更新使用的路由,以请求头和host作为路由匹配规则
		// 先写入路由表，避免多协程对单个路由多次注册
		r.table.Store(key, ticker)
		route, err := r.gateway.Create(&kong.RouteCreateOption{
			Name:      key,
			Protocols: []string{"http"},
			Methods:   []string{"POST"},
			Hosts:     []string{"gd-k8s-master01"},
			Headers: map[string][]string{
				"X-Device-ID": {key},
			},
			StripPath: false,
			Service: &struct {
				Name string `json:"name,omitempty"`
				Id   string `json:"id,omitempty"`
			}{Name: r.serviceName},
		})
		if err != nil {
			return err
		}
		ticker := time.NewTicker(r.timeout)
		go r.autoUnRegister(ticker, route)
		return nil
	}
}

// UnRegister 注销路由
func (r *RouteRegister) UnRegister(info *DeviceGeneralInfo) {
	key := fmt.Sprintf("%s_%d_%s", service.Username, info.DeviceClassID, info.DeviceID)
	if ticker, ok := r.table.Load(key); ok {
		// 通过设置定时器为1纳秒，快速触发路由的自动注销
		ticker.(*time.Ticker).Reset(time.Nanosecond)
	}
}
func (r *RouteRegister) autoUnRegister(ticker *time.Ticker, route kong.Object) {
	<-ticker.C
	// 注销路由并删除路由表中信息
	// 先向网关注销路由，再删除路由表中信息，
	// 确保先注销路由，避免向网关注册路由和注销路由同时发生
	r.gateway.Delete(route)
	r.table.Delete(route.(*kong.Route).Id)
}
