package router

import (
	"encoding/json"
	"fmt"

	"github.com/cykyes/flupoc-go/poculum"
)

// HandlerFunc 路由处理函数类型
// 返回响应数据和可能的错误
// 中间件应调用 next 并返回其结果。
type HandlerFunc func(*Context) (*Response, error)

// Middleware 定义中间件，包装下游处理器
// 允许在调用前后执行逻辑。
type Middleware func(HandlerFunc) HandlerFunc

// Request 表示路由分发所需的最小请求信息
// Path 可以包含查询串，例如 /users/1?page=2
type Request struct {
	Method string
	Path   string
	Body   []byte
}

// Context 请求上下文，包含路由处理所需的所有信息
// 未来可扩展：Method、Path、RequestBody 等
// 目前仅包含路径参数和查询参数。
type Context struct {
	PathParams  map[string]string
	QueryParams map[string]string
	RequestBody []byte // 原始请求体
}

// Response 响应数据结构
// Body 可以是字符串、[]byte、结构体等。
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       interface{}
}

// NewResponse 创建一个默认的成功响应
func NewResponse(body interface{}) *Response {
	return &Response{
		StatusCode: 200,
		Headers:    make(map[string]string),
		Body:       body,
	}
}

// NewTextResponse 创建用于UTF-8文本的响应，自动设置Content-Type
func NewTextResponse(text string) *Response {
	resp := NewResponse(text)
	resp.Headers["Content-Type"] = "utf-8"
	return resp
}

// NewBytesResponse 创建用于字节流的响应，自动设置Content-Type
func NewBytesResponse(data []byte) *Response {
	resp := NewResponse(data)
	resp.Headers["Content-Type"] = "bytes"
	return resp
}

// Bytes 将响应体转换为字节流
func (resp *Response) Bytes() ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	switch body := resp.Body.(type) {
	case nil:
		return []byte{}, nil
	case []byte:
		return body, nil
	case string:
		return []byte(body), nil
	default:
		return json.Marshal(body)
	}
}

// BytesToRequest 使用 Poculum 将二进制载荷解码为 Request
func BytesToRequest(data []byte) (*Request, error) {
	decoded, err := poculum.LoadPoculum(data)
	if err != nil {
		return nil, fmt.Errorf("decode request: %w", err)
	}

	reqMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("decoded request is %T, want map[string]any", decoded)
	}

	method, ok := reqMap["method"].(string)
	if !ok || method == "" {
		return nil, fmt.Errorf("request method missing or not string")
	}

	path, ok := reqMap["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("request path missing or not string")
	}

	var body []byte
	if v, exists := reqMap["body"]; exists {
		switch b := v.(type) {
		case nil:
			body = nil
		case []byte:
			body = b
		default:
			return nil, fmt.Errorf("unsupported body type %T: body must be []byte", b)
		}
	}

	return &Request{Method: method, Path: path, Body: body}, nil
}

// ResponseToBytes 使用 Poculum 将 Response 序列化为二进制
func ResponseToBytes(resp *Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("response is nil")
	}

	payload := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Headers,
		"body":    resp.Body,
	}

	return poculum.DumpPoculum(payload)
}
