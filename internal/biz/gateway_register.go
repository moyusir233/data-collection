package biz

import (
	"fmt"
	"gitee.com/moyusir/dataCollection/internal/conf"
	"gitee.com/moyusir/util/kong"
	"github.com/go-kratos/kratos/v2/log"
	"sync"
	"time"
)

// GatewayRegister 负责路由的注册以及自动注销
type GatewayRegister struct {
	// 网关客户端
	gateway *kong.Admin
	// 路由的自动注销时间
	timeout time.Duration
	// route注册表
	table *sync.Map
}

func NewGatewayRegister(c *conf.Server, logger log.Logger) *GatewayRegister {
	return &GatewayRegister{
		gateway: kong.NewAdmin(c.Gateway.Address, logger),
		timeout: c.Gateway.RouteTimeout.AsDuration(),
		table:   new(sync.Map),
	}
}

// CreateService 创建service对象
func (r *GatewayRegister) CreateService() error {

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
			}{Name: conf.ServiceName},
		})
		if err != nil {
			return err
		}
		ticker := time.NewTicker(r.timeout)
		go r.autoUnRegister(ticker, route)
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
