package kong

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

// Service kong中的服务对象，代表着集群中的微服务
type Service struct {
	Id                string   `json:"id,omitempty"`
	Name              string   `json:"name,omitempty"`
	Retries           int      `json:"retries,omitempty"`
	Protocol          string   `json:"protocol,omitempty"`
	Host              string   `json:"host,omitempty"`
	Port              int      `json:"port,omitempty"`
	Path              string   `json:"path,omitempty"`
	ConnectTimeout    int      `json:"connect_timeout,omitempty"`
	WriteTimeout      int      `json:"write_timeout,omitempty"`
	ReadTimeout       int      `json:"read_timeout,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	ClientCertificate *struct {
		Id string `json:"id,omitempty"`
	} `json:"client_certificate,omitempty"`
	TlsVerify      bool        `json:"tls_verify,omitempty"`
	TlsVerifyDepth interface{} `json:"tls_verify_depth,omitempty"`
	CaCertificates []string    `json:"ca_certificates,omitempty"`
	Enabled        bool        `json:"enabled,omitempty"`
	// http客户端
	Client *req.Client `json:"-"`
}

// ServiceCreateOption 创建服务的常用选项
type ServiceCreateOption struct {
	// 唯一标识服务的名称
	Name string `json:"name,omitempty"`
	// 可用的协议类型，
	// 包括"grpc", "grpcs", "http", "https", "tcp", "tls", "tls_passthrough", "udp"，
	// 默认使用http
	Protocol string `json:"protocol,omitempty"`
	// 服务的主机地址，注意是大小写敏感的
	Host string `json:"host,omitempty"`
	// 服务端口，默认80
	Port int `json:"port,omitempty"`
	// 请求服务的路径
	Path string `json:"path,omitempty"`
	// 是否启用服务，若不启用，则请求任意与该服务相关联的route都会返回404
	Enabled bool `json:"enabled,omitempty"`
	// 一次设置协议、主机、端口和路径的简写属性。
	Url string `json:"url,omitempty"`
	// tag
	Tags []string `json:"tags,omitempty"`
}

func (s *Service) create(client *req.Client, option interface{}) error {
	s.Client = client
	o := option.(*ServiceCreateOption)
	path := "/services"
	request := client.R().
		SetBodyJsonMarshal(o).
		SetResult(s)
	if err := sendRequest(request, http.MethodPost, path); err != nil {
		return errors.Newf(
			500, "SERVICE_CREATE_FAIL", "服务:%s创建失败\n错误信息:%s", o.Name, err)
	}
	return nil
}
func (s *Service) delete() error {
	path := "/services/{id}"
	request := s.Client.R().
		SetPathParam("id", s.Id)
	if err := sendRequest(request, http.MethodDelete, path); err != nil {
		return errors.Newf(
			500, "SERVICE_DELETE_FAIL", "服务:%s删除失败\n错误信息:%s", s.Name, err)
	}
	return nil
}
