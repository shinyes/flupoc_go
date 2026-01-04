package router

import (
	"fmt"

	"github.com/cykyes/flupoc-go/poculum"
)

// ExampleUsage 展示如何使用 Router 结合 Poculum 做二进制编解码
func ExampleUsage() {
	r := NewRouter()

	// 简单的回显路由
	r.Post("/echo", func(ctx *Context) (*Response, error) {
		return NewBytesResponse(ctx.RequestBody), nil
	})

	// 构造 Poculum 序列化后的请求报文
	payload := map[string]any{
		"method": "POST",
		"path":   "/echo",
		"body":   []byte("hello poculum"),
	}
	encodedReq, err := poculum.DumpPoculum(payload)
	if err != nil {
		fmt.Println("序列化请求失败:", err)
		return
	}

	// 通过 BytesToRequest 反序列化为内部 Request 结构
	req, err := BytesToRequest(encodedReq)
	if err != nil {
		fmt.Println("反序列化请求失败:", err)
		return
	}

	// 走路由分发
	resp, err := r.ServeRequest(req)
	if err != nil {
		fmt.Println("路由处理失败:", err)
		return
	}

	// 将 Response 再次用 Poculum 序列化
	encodedResp, err := ResponseToBytes(resp)
	if err != nil {
		fmt.Println("序列化响应失败:", err)
		return
	}

	// 演示反序列化查看数据
	decoded, err := poculum.LoadPoculum(encodedResp)
	if err != nil {
		fmt.Println("反序列化响应失败:", err)
		return
	}

	m, ok := decoded.(map[string]any)
	if !ok {
		fmt.Println("响应结构错误")
		return
	}

	// 输出关键字段，便于文档示例
	statusAny := m["status"]
	bodyAny := m["body"]
	var status uint32
	switch v := statusAny.(type) {
	case uint32:
		status = v
	case uint64:
		status = uint32(v)
	case int64:
		status = uint32(v)
	}
	var bodyStr string
	if b, ok := bodyAny.([]byte); ok {
		bodyStr = string(b)
	}

	fmt.Println("status:", status)
	fmt.Println("body:", bodyStr)

	// Output:
	// status: 200
	// body: hello poculum
}
