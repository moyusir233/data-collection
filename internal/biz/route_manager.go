package biz

import (
	"context"
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

const DELETED_TAG = "deleted_tag"

// RouteManager 负责管理路由，包括api网关的路由以及服务内部关于设备更新连接的路由
type RouteManager struct {
	// 网关客户端
	gateway *kong.Admin
	// 路由的自动注销时间
	timeout time.Duration
	// route注册表
	table *RouteTable
	// 用于多协程控制
	eg   *errgroup.Group
	ctx  context.Context
	done context.CancelFunc
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

	ctx, cancel := context.WithCancel(context.Background())

	return &RouteManager{
		gateway: kong.NewAdmin(c.Gateway.Address),
		timeout: c.Gateway.RouteTimeout.AsDuration(),
		table:   new(RouteTable),
		eg:      new(errgroup.Group),
		ctx:     ctx,
		done:    cancel,
	}
}

// Init 创建服务的网关组件service以及plugin,route动态创建
func (r *RouteManager) Init() error {
	options := []interface{}{
		defaultServiceCreateOption,
		defaultAuthPluginCreateOption,
	}
	for _, o := range options {
		_, err := r.gateway.Create(o)
		if err != nil {
			return err
		}
		// 组件创建需要间隔一段时间
		time.Sleep(500 * time.Millisecond)
	}
	return nil
}

// Close 清理服务注册的相关网关组件
func (r *RouteManager) Close() error {
	// 依据注册时的tag将所有路由组件统一删除
	r.gateway.Clear(kong.FLAG_ROUTE|kong.FLAG_PLUGIN|kong.FLAG_SERVICE, conf.ServiceName)
	// 遍历路由表，寻找父节点，进而关闭channel、定时器以及负责自动注销的协程
	r.table.Range(func(key, value interface{}) bool {
		node := value.(*RouteTableNode)
		if parent := r.table.Find(node); parent.RouteTag != DELETED_TAG {
			parent.UnregisterTicker.Reset(time.Nanosecond)
			close(parent.UpdateChannel)
			parent.RouteTag = DELETED_TAG
		}
		return true
	})
	// 等待所有路由都被注销
	r.done()
	r.eg.Wait()
	return nil
}

// LoadOrCreateParentNode 查询或创建用户ID对应的，保存相应路由信息的父节点，
func (r *RouteManager) LoadOrCreateParentNode(clientID string) *RouteTableNode {
	// 查询clientID对应父节点，若存在则进行复用，不存在则初始化父节点
	if node, ok := r.table.Load(clientID); ok {
		return node.(*RouteTableNode)
	} else {
		root := new(RouteTableNode)
		root.UnregisterTicker = time.NewTicker(r.timeout)
		// TODO 考虑更新channel的容量问题
		root.UpdateChannel = make(chan interface{}, 5)
		root.RouteTag = clientID
		r.eg.Go(func() error {
			r.autoUnRegister(root)
			return nil
		})
		r.table.Store(clientID, root)
		return root
	}
}

// GetDeviceUpdateChannel 查找设备对应的配置更新channel
func (r *RouteManager) GetDeviceUpdateChannel(info *DeviceGeneralInfo) (chan<- interface{}, bool) {
	node, ok := r.table.Load(GetKey(info))
	if ok {
		return r.table.Find(node.(*RouteTableNode)).UpdateChannel, true
	} else {
		return nil, false
	}
}

// ActivateRoute 激活给定设备的路由,clientID为客户端连接标识符
// 函数会为clientID创建相应的路由表节点，称为父节点(协程父节点不代表任何设备，只是用于保存路由资源)
// 当clientID对应的路由表节点为nil，即协程父节点还未初始化时，函数结合传入的info初始化协程父节点
func (r *RouteManager) ActivateRoute(clientID string, info *DeviceGeneralInfo) error {
	// 检索clientID对应的父节点
	var root *RouteTableNode
	if n, ok := r.table.Load(clientID); ok {
		root = n.(*RouteTableNode)
	}

	key := GetKey(info)
	var err error
	if n, ok := r.table.Load(key); ok {
		node := n.(*RouteTableNode)
		parent := r.table.Find(node)
		if root == nil {
			// 父节点不存在，而key对应节点的存在，则复用key对应节点的父节点
			root = parent
			r.table.Store(clientID, root)
		} else {
			// 当父节点与key对应节点都存在时，通过将key对应节点与root进行连接以及修改tag,
			// 实现将key对应节点的路由信息更新
			// (注意这里并没有直接将key对应的节点群整个连接到root上,只是对单个设备的路由进行了更新)
			if root.RouteTag != parent.RouteTag {
				// node只是个未连接的普通节点，则修改其连接关系和tag
				r.table.Join(root, node)
				(&kong.Route{Name: key, Client: r.gateway.Client}).Update(&kong.RouteCreateOption{
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
					Tags:      append(defaultRouteCreateOption.Tags, (*root).RouteTag),
				})
			}
		}
		return nil
	} else {
		// 为key实例化对应的路由节点
		node := new(RouteTableNode)
		r.table.Store(key, node)

		if root == nil {
			// 父节点为空，则初始化一个存储路由资源的父节点
			root = r.LoadOrCreateParentNode(clientID)
		}
		// 连接到root节点上
		r.table.Join(root, node)
		// 依据root的tag,为新路由节点创建路由信息
		option := &kong.RouteCreateOption{
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
			Tags:      append(defaultRouteCreateOption.Tags, (*root).RouteTag),
		}
		route, createErr := r.gateway.Create(option)
		err = createErr
		// 路由的创建失败大部分原因下是由于负载均衡导致客户端在多个容器服务处
		// 注册了路由信息，造成路由创建冲突，此时更新相应的路由信息即可
		if err != nil {
			r := route.(*kong.Route)
			r.Name = key
			r.Update(option)
		}
	}
	// 重置定时器，相当于激活路由
	(*root).UnregisterTicker.Reset(r.timeout)
	return err
}

// UnRegisterRoute 注销路由，包括网关路由信息与路由表中信息(将相关联的route组统一注销，用于客户端正常断联时)
func (r *RouteManager) UnRegisterRoute(info *DeviceGeneralInfo) {
	key := GetKey(info)
	if n, ok := r.table.Load(key); ok {
		parent := r.table.Find(n.(*RouteTableNode))
		// 清除路由资源
		// 通过设置定时器为1纳秒，快速触发路由的自动注销
		parent.UnregisterTicker.Reset(time.Nanosecond)
		close(parent.UpdateChannel)
		// 清除路由表中的信息
		var children []string
		r.table.Range(func(key, value interface{}) bool {
			if r.table.Find(value.(*RouteTableNode)) == parent {
				children = append(children, key.(string))
			}
			return true
		})
		for _, k := range children {
			r.table.Delete(k)
		}
	}
}

// 自动注销仅仅是将节点对应的若干route信息注销，并将节点的RouteTag标志为已删除的tag
// 而不会删除节点在路由表中存储的信息，只有当协程检测到客户端的连接正常断开时
// 才会进行节点的路由注销以及节点信息的删除操作，包括关闭其配置更新channel等
// TODO 考虑定时器函数是否需要清理路由表中的键值对信息
func (r *RouteManager) autoUnRegister(node *RouteTableNode) {
	select {
	case <-node.UnregisterTicker.C:
	case <-r.ctx.Done():
	}
	if node.RouteTag != DELETED_TAG {
		r.gateway.Clear(kong.FLAG_ROUTE, node.RouteTag)
		node.RouteTag = DELETED_TAG
	}
	node.UnregisterTicker.Stop()
}

// GetKey 辅助函数
func GetKey(info *DeviceGeneralInfo) string {
	return fmt.Sprintf("%s_%d_%s", conf.Username, info.DeviceClassID, info.DeviceID)
}
