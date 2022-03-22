package kong

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

// Route 路由代表着对前端请求的转发规则，每个服务通常对应着单个或多个路由
type Route struct {
	Id   string `json:"id,omitempty"`
	Name string `json:"name,omitempty"`
	//Protocols               []string            `json:"protocols,omitempty"`
	//Methods                 []string            `json:"methods,omitempty"`
	//Hosts                   []string            `json:"hosts,omitempty"`
	//Paths                   []string            `json:"paths,omitempty"`
	//Headers                 map[string][]string `json:"headers,omitempty"`
	//HttpsRedirectStatusCode int                 `json:"https_redirect_status_code,omitempty"`
	//RegexPriority           int                 `json:"regex_priority,omitempty"`
	//StripPath               bool                `json:"strip_path,omitempty"`
	//PathHandling            string              `json:"path_handling,omitempty"`
	//PreserveHost            bool                `json:"preserve_host,omitempty"`
	//RequestBuffering        bool                `json:"request_buffering,omitempty"`
	//ResponseBuffering       bool                `json:"response_buffering,omitempty"`
	//Tags                    []string            `json:"tags,omitempty"`
	//Service                 struct {
	//	Id string `json:"id,omitempty"`
	//} `json:"service,omitempty"`
	// http客户端
	Client *req.Client `json:"-"`
}

// RouteCreateOption 创建路由的常用选项
type RouteCreateOption struct {
	// route的唯一标识名称
	Name string `json:"name,omitempty"`
	// 该route允许的协议列表
	Protocols []string `json:"protocols,omitempty"`
	// route匹配的http方法
	Methods []string `json:"methods,omitempty"`
	// route匹配的主机地址
	Hosts []string `json:"hosts,omitempty"`
	// route匹配的路径
	Paths []string `json:"paths,omitempty"`
	// route匹配的请求头
	Headers map[string][]string `json:"headers,omitempty"`
	// 通过其中一个路径匹配 Route 时，从上游请求 URL 中去除匹配的前缀,默认为true
	StripPath bool `json:"strip_path"`
	// 与该route关联的service
	Service *struct {
		Name string `json:"name,omitempty"`
		Id   string `json:"id,omitempty"`
	} `json:"service"`
	// tag
	Tags []string `json:"tags,omitempty"`
}

func (r *Route) create(client *req.Client, option interface{}) error {
	r.Client = client
	o := option.(*RouteCreateOption)
	path := "/routes"
	request := client.R().
		SetBodyJsonMarshal(option).
		SetResult(r)
	if err := sendRequest(request, http.MethodPost, path); err != nil {
		return errors.Newf(
			500, "ROUTE_CREATE_FAIL", "路由:%s创建失败\n错误信息:%s", o.Name, err)
	}
	return nil
}
func (r *Route) delete() error {
	path := "/routes/{id}"
	request := r.Client.R().
		SetPathParam("id", r.Id)
	if err := sendRequest(request, http.MethodDelete, path); err != nil {
		return errors.Newf(
			500, "ROUTE_DELETE_FAIL", "路由:%s删除失败\n错误信息:%s", r.Name, err)
	}
	return nil
}
func (r *Route) Update(option *RouteCreateOption) error {
	request := r.Client.R().
		SetBodyJsonMarshal(option).
		SetResult(r)
	var path string
	if r.Id != "" {
		request.SetPathParam("id", r.Id)
		path = "/routes/{id}"
	} else {
		request.SetPathParam("name", r.Name)
		path = "/routes/{name}"
	}
	if err := sendRequest(request, http.MethodPatch, path); err != nil {
		return errors.Newf(
			500, "ROUTE_UPDATE_FAIL", "路由:%s更新失败\n错误信息:%s", r.Name, err)
	}
	return nil
}
