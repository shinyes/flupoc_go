package router

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/cykyes/flupoc-go/poculum"
)

// HandlerFunc 定义路由处理函数签名。
// 接收 Context，返回 Response 与可选错误。
type HandlerFunc func(*Context) (*Response, error)

// Middleware 包裹处理函数以添加前置/后置逻辑。
type Middleware func(HandlerFunc) HandlerFunc

// Request 表示入站请求。
type Request struct {
	Method string
	Path   string
	Body   []byte
}

// Context 在处理链中携带请求级数据。
type Context struct {
	context.Context // 嵌入标准 context 以支持取消/超时

	PathParams  map[string]string
	QueryParams map[string]string
	RequestBody []byte

	// Request metadata (populated by server)
	Method string
	Path   string
}

// NewContext 基于给定上下文创建新的 Context。
func NewContext(ctx context.Context) *Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return &Context{
		Context:     ctx,
		PathParams:  make(map[string]string),
		QueryParams: make(map[string]string),
	}
}

// Param 返回路径参数的值。
func (c *Context) Param(key string) string {
	if c.PathParams == nil {
		return ""
	}
	return c.PathParams[key]
}

// Query 返回查询参数的值。
func (c *Context) Query(key string) string {
	if c.QueryParams == nil {
		return ""
	}
	return c.QueryParams[key]
}

// Response 表示一条出站响应。
type Response struct {
	StatusCode int
	Headers    map[string]string
	Body       []byte
}

// NewResponse 使用给定 Body 创建 200 OK 响应。
func NewResponse(body []byte) *Response {
	return &Response{
		StatusCode: 200,
		Headers:    make(map[string]string),
		Body:       body,
	}
}

// OK 创建带可选 Body 的 200 响应。
func OK(body []byte) *Response {
	return NewResponse(body)
}

// Error 创建指定状态码和消息的错误响应。
func Error(statusCode int, message string) *Response {
	return &Response{
		StatusCode: statusCode,
		Headers:    map[string]string{"Content-Type": "text/plain"},
		Body:       []byte(message),
	}
}

// JSON 创建带 JSON 内容类型的响应。
// 当 body 无法序列化为 JSON 时，返回 500 错误响应。
func JSON(body interface{}) *Response {
	data, err := json.Marshal(body)
	if err != nil {
		return &Response{
			StatusCode: 500,
			Headers:    map[string]string{"Content-Type": "application/json"},
			Body:       []byte(`{"error":"json marshal failed"}`),
		}
	}
	resp := NewResponse(data)
	resp.Headers["Content-Type"] = "application/json"
	return resp
}

// Text 创建纯文本响应。
func Text(text string) *Response {
	return &Response{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "text/plain; charset=utf-8"},
		Body:       []byte(text),
	}
}

// Bytes 创建二进制响应。
func Bytes(data []byte) *Response {
	return &Response{
		StatusCode: 200,
		Headers:    map[string]string{"Content-Type": "application/octet-stream"},
		Body:       data,
	}
}

// WithStatus 设置状态码并返回响应以便链式调用。
func (r *Response) WithStatus(code int) *Response {
	r.StatusCode = code
	return r
}

// WithHeader 添加 Header 并返回响应以便链式调用。
func (r *Response) WithHeader(key, value string) *Response {
	if r.Headers == nil {
		r.Headers = make(map[string]string)
	}
	r.Headers[key] = value
	return r
}

// --- 已弃用的兼容函数 ---

// NewTextResponse 已弃用，使用 Text。
func NewTextResponse(text string) *Response {
	return Text(text)
}

// NewBytesResponse 已弃用，使用 Bytes。
func NewBytesResponse(data []byte) *Response {
	return Bytes(data)
}

// GetBody 返回响应体的字节数据。
func (resp *Response) GetBody() ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应为空")
	}
	if resp.Body == nil {
		return []byte{}, nil
	}
	return resp.Body, nil
}

// BytesToRequest decodes a Poculum-encoded request.
func BytesToRequest(data []byte) (*Request, error) {
	decoded, err := poculum.LoadPoculum(data)
	if err != nil {
		return nil, fmt.Errorf("解码请求: %w", err)
	}

	reqMap, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("解码结果为 %T，期望 map[string]any", decoded)
	}

	method, ok := reqMap["method"].(string)
	if !ok || method == "" {
		return nil, fmt.Errorf("请求 method 缺失或不是字符串")
	}

	path, ok := reqMap["path"].(string)
	if !ok || path == "" {
		return nil, fmt.Errorf("请求 path 缺失或不是字符串")
	}

	var body []byte
	if v, exists := reqMap["body"]; exists {
		switch b := v.(type) {
		case nil:
			body = nil
		case []byte:
			body = b
		default:
			return nil, fmt.Errorf("不支持的 body 类型 %T: body 必须为 []byte", b)
		}
	}

	return &Request{Method: method, Path: path, Body: body}, nil
}

// ResponseToBytes encodes a response using Poculum.
func ResponseToBytes(resp *Response) ([]byte, error) {
	if resp == nil {
		return nil, fmt.Errorf("响应为空")
	}

	payload := map[string]any{
		"status":  resp.StatusCode,
		"headers": resp.Headers,
		"body":    resp.Body,
	}

	return poculum.DumpPoculum(payload)
}
