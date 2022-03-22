package kong

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

// Consumer consumer代表服务的消费者或用户
type Consumer struct {
	Id       string   `json:"id"`
	Username string   `json:"username"`
	CustomId string   `json:"custom_id"`
	Tags     []string `json:"tags"`
	// http客户端
	Client *req.Client `json:"-"`
}
type ConsumerCreateOption struct {
	// consumer的唯一标识名称
	Username string `json:"username,omitempty"`
	// tag
	Tags []string `json:"tags,omitempty"`
}

func (c *Consumer) create(client *req.Client, option interface{}) error {
	c.Client = client
	o := option.(*ConsumerCreateOption)
	path := "/consumers"
	request := client.R().
		SetBodyJsonMarshal(option).
		SetResult(c)
	if err := sendRequest(request, http.MethodPost, path); err != nil {
		return errors.Newf(
			500, "CONSUMER_CREATE_FAIL", "消费者:%s创建失败\n错误信息:%s", o.Username, err)
	}
	return nil
}
func (c *Consumer) delete() error {
	request := c.Client.R().SetPathParam("id", c.Id)
	path := "/consumers/{id}"
	if err := sendRequest(request, http.MethodDelete, path); err != nil {
		return errors.Newf(
			500, "CONSUMER_DELETE_FAIL", "消费者:%s删除失败\n错误信息:%s", c.Username, err)
	}
	return nil
}
