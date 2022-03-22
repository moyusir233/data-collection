package kong

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

var PluginsName = map[string]string{
	"key":      "key-auth",
	"response": "response-transformer",
}

// Plugin plugin实体表示将在 HTTP 请求/响应生命周期中执行的插件配置。
type Plugin struct {
	Id       string    `json:"id"`
	Name     string    `json:"name"`
	Route    *Route    `json:"route"`
	Service  *Service  `json:"service"`
	Consumer *Consumer `json:"consumer"`
	Config   struct {
		Minute int `json:"minute"`
		Hour   int `json:"hour"`
	} `json:"config"`
	Protocols []string `json:"protocols"`
	Enabled   bool     `json:"enabled"`
	Tags      []string `json:"tags"`
	// http客户端
	Client *req.Client `json:"-"`
}

// KeyAuthPluginCreateOption 创建选项，若service、route与consumer对象都为空，则建立全局插件
type KeyAuthPluginCreateOption struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Service *struct {
		Name string `json:"name,omitempty"`
		Id   string `json:"id,omitempty"`
	} `json:"service"`
	Route *struct {
		Name string `json:"name,omitempty"`
		Id   string `json:"id,omitempty"`
	} `json:"route"`
	Consumer *struct {
		Username string `json:"username,omitempty"`
		Id       string `json:"id,omitempty"`
	} `json:"consumer"`
	Config *KeyAuthPluginConfig `json:"config"`
	// tag
	Tags []string `json:"tags,omitempty"`
}
type KeyAuthPluginConfig struct {
	HideCredentials bool     `json:"hide_credentials,omitempty"`
	KeyNames        []string `json:"key_names,omitempty"`
	KeyInBody       bool     `json:"key_in_body,omitempty"`
	KeyInHeader     bool     `json:"key_in_header,omitempty"`
	KeyInQuery      bool     `json:"key_in_query,omitempty"`
}

// ResponseTransformerPluginCreateOption 修改响应报文的插件，也可以通过不指定相关的对象完成全局启用
type ResponseTransformerPluginCreateOption struct {
	Name    string `json:"name,omitempty"`
	Enabled bool   `json:"enabled,omitempty"`
	Service *struct {
		Id string `json:"id,omitempty"`
	} `json:"service,omitempty"`
	Route *struct {
		Id string `json:"id,omitempty"`
	} `json:"route,omitempty"`
	Consumer *struct {
		Id string `json:"id,omitempty"`
	} `json:"consumer,omitempty"`
	Config *ResponseTransformerPluginConfig `json:"config"`
	// tag
	Tags []string `json:"tags,omitempty"`
}
type ResponseTransformerPluginConfig struct {
	Add *struct {
		// 以headername:value组成的字符串描述每个要填加的响应头以及值
		Headers []string `json:"headers,omitempty"`
	} `json:"add"`
}

func (p *Plugin) create(client *req.Client, option interface{}) error {
	p.Client = client
	var name string
	switch option.(type) {
	case *KeyAuthPluginCreateOption:
		name = PluginsName["key"]
		option.(*KeyAuthPluginCreateOption).Name = name
	case *ResponseTransformerPluginCreateOption:
		name = PluginsName["response"]
		option.(*ResponseTransformerPluginCreateOption).Name = name
	default:
		return errors.New(500, "PLUGIN_CREATE_FAIL", "暂不支持的插件类型")
	}
	request := client.R().
		SetBodyJsonMarshal(option).
		SetResult(p)
	path := "/plugins"
	if err := sendRequest(request, http.MethodPost, path); err != nil {
		return errors.Newf(
			500, "PLUGIN_CREATE_FAIL", "插件:%s创建失败\n错误信息:%s", name, err)
	}
	return nil
}

func (p *Plugin) delete() error {
	request := p.Client.R().SetPathParam("id", p.Id)
	path := "/plugins/{id}"
	if err := sendRequest(request, http.MethodDelete, path); err != nil {
		return errors.Newf(
			500, "PLUGIN_DELETE_FAIL", "插件:%s创建失败\n错误信息:%s", p.Name, err)
	}
	return nil
}
