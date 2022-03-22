package kong

import (
	v1 "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
	"strings"
)

const (
	FLAG_SERVICE = 1 << iota
	FLAG_ROUTE
	FLAG_PLUGIN
	FLAG_CONSUMER
)

// Admin 通过kong提供的admin api管理kong网关
type Admin struct {
	// kong网关的http客户端
	Client *req.Client
}

// Object kong对象接口
type Object interface {
	// 创建对象
	create(client *req.Client, option interface{}) error
	// 删除对象
	delete() error
}

// NewAdmin 构造器函数
func NewAdmin(address string) (*Admin, error) {
	k := &Admin{
		Client: req.C().SetBaseURL(address).EnableKeepAlives(),
	}
	if err := k.Ping(); err != nil {
		return nil, err
	}
	return k, nil
}

// Ping 测试网关是否连通
func (admin *Admin) Ping() error {
	if err := sendRequest(admin.Client.R(), http.MethodGet, ""); err != nil {
		return v1.ErrorApiGatewayConnectFail("failed to connect gateway,msg:%s", err)
	}
	return nil
}

// Create 依据option类型创建相应的kong对象
func (admin *Admin) Create(option interface{}) (object Object, err error) {
	switch option.(type) {
	case *ServiceCreateOption:
		object = new(Service)
	case *RouteCreateOption:
		object = new(Route)
	case *ConsumerCreateOption:
		object = new(Consumer)
	case *KeyAuthPluginCreateOption:
		object = new(Plugin)
	case *ResponseTransformerPluginCreateOption:
		object = new(Plugin)
	case *KeyCreateOption:
		object = new(Key)
	default:
		return nil, errors.New(500, "KONG_OBJECT_CREATE_FAIL", "kong object does not exist")
	}
	if err = object.create(admin.Client, option); err != nil {
		return object, err
	}
	return
}

// Delete 删除kong对象
func (admin *Admin) Delete(object Object) error {
	if err := object.delete(); err != nil {
		return err
	}
	return nil
}

// Clear 依据tag清除所有的服务实体,
// 从低位到高位，flag的低四位分别标识着service、route、plugin、consumer 标识是否需要清除
func (admin *Admin) Clear(flag byte, tags ...string) {
	// 将查询结果的json字符串封装到该结构体中
	data := &struct {
		Data []struct {
			Id string `json:"id"`
		} `json:"data"`
		Next interface{} `json:"next"`
	}{}
	t := strings.Join(tags, ",")

	request := admin.Client.R().SetQueryParam("tags", t)
	r := request.SetResult(data)
	if (flag & FLAG_PLUGIN) == FLAG_PLUGIN {
		sendRequest(r, http.MethodGet, "/plugins")
		for _, d := range data.Data {
			admin.Delete(&Plugin{Id: d.Id, Client: admin.Client})
		}
	}
	if (flag & FLAG_CONSUMER) == FLAG_CONSUMER {
		sendRequest(r, http.MethodGet, "/consumers")
		for _, d := range data.Data {
			admin.Delete(&Consumer{Id: d.Id, Client: admin.Client})
		}
	}
	if (flag & FLAG_ROUTE) == FLAG_ROUTE {
		sendRequest(r, http.MethodGet, "/routes")
		for _, d := range data.Data {
			admin.Delete(&Route{Id: d.Id, Client: admin.Client})
		}
	}
	if (flag & FLAG_SERVICE) == FLAG_SERVICE {
		sendRequest(r, http.MethodGet, "/services")
		for _, d := range data.Data {
			admin.Delete(&Service{Id: d.Id, Client: admin.Client})
		}
	}
}
