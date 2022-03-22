package kong

import (
	v1 "gitee.com/moyusir/util/api/util/v1"
	"github.com/go-kratos/kratos/v2/errors"
	"github.com/imroc/req/v3"
	"net/http"
)

// ResponseError 描述创建或删除kong对象时遇到的错误
type ResponseError struct {
	Body string
}

// UnmarshalJSON ResponseError直接保存utf8序列化后的字符串信息
func (e *ResponseError) UnmarshalJSON(bytes []byte) error {
	e.Body = string(bytes)
	return nil
}

func (e *ResponseError) Error() string {
	return e.Body
}

// 请求发送的包装函数,以进行统一的错误处理
func sendRequest(request *req.Request, method string, path string) (err error) {
	var (
		resp    *req.Response
		respErr = new(ResponseError)
	)
	request.SetError(respErr)
	switch method {
	case http.MethodGet:
		resp, err = request.Get(path)
	case http.MethodPost:
		resp, err = request.Post(path)
	case http.MethodDelete:
		resp, err = request.Delete(path)
	case http.MethodHead:
		resp, err = request.Head(path)
	case http.MethodOptions:
		resp, err = request.Options(path)
	case http.MethodPatch:
		resp, err = request.Patch(path)
	default:
		return errors.New(500, "UnsupportedMethod", "Unsupported Method")
	}
	if err != nil {
		return v1.ErrorRequestSendFail("failed to send http request，msg:%s", err)
	}
	if resp.IsError() {
		return respErr
	}
	return nil
}
