package kong

import (
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

// Key key是key auth插件中，consumer用于访问route或service的凭证信息
type Key struct {
	Consumer struct {
		Id string `json:"id"`
	} `json:"consumer"`
	Id  string `json:"id"`
	Key string `json:"key"`
	// http客户端
	Client *req.Client `json:"-"`
}
type KeyCreateOption struct {
	// 通过指定consumer的username创建相应的key
	Username string
}

func (k *Key) create(client *req.Client, option interface{}) error {
	k.Client = client
	o := option.(*KeyCreateOption)
	request := client.R().SetResult(k).SetPathParam("username", o.Username)
	path := "/consumers/{username}/key-auth"
	if err := sendRequest(request, http.MethodPost, path); err != nil {
		return errors.Newf(
			500, "KEY_CREATE_FAIL",
			"用户:%s的key创建失败\n错误信息:%s", o.Username, err)
	}
	return nil
}

func (k *Key) delete() error {
	request := k.Client.R().SetPathParams(map[string]string{
		"cid": k.Consumer.Id,
		"kid": k.Id,
	})
	path := "/consumers/{cid}/key-auth/{kid}"
	if err := sendRequest(request, http.MethodDelete, path); err != nil {
		return errors.Newf(
			500, "KEY_DELETE_FAIL", "用户:%s的key删除失败\n错误信息:%s", k.Consumer.Id, err)
	}
	return nil
}
