package router

import (
	"reflect"
	"testing"

	"github.com/cykyes/flupoc-go/poculum"
)

func TestBytesToRequest(t *testing.T) {
	t.Run("包含body且为bytes", func(t *testing.T) {
		payload := map[string]any{
			"method": "POST",
			"path":   "/echo",
			"body":   []byte("hello"),
		}

		encoded, err := poculum.DumpPoculum(payload)
		if err != nil {
			t.Fatalf("序列化payload失败: %v", err)
		}

		req, err := BytesToRequest(encoded)
		if err != nil {
			t.Fatalf("反序列化请求失败: %v", err)
		}

		if req.Method != "POST" || req.Path != "/echo" {
			t.Fatalf("请求基本字段不匹配: %+v", req)
		}

		if string(req.Body) != "hello" {
			t.Fatalf("请求体不匹配，期望 hello，实际 %s", string(req.Body))
		}
	})

	t.Run("缺少body键时为nil", func(t *testing.T) {
		payload := map[string]any{
			"method": "GET",
			"path":   "/ping",
		}

		encoded, err := poculum.DumpPoculum(payload)
		if err != nil {
			t.Fatalf("序列化payload失败: %v", err)
		}

		req, err := BytesToRequest(encoded)
		if err != nil {
			t.Fatalf("反序列化请求失败: %v", err)
		}

		if req.Body != nil {
			t.Fatalf("期望 Body 为 nil，实际长度 %d", len(req.Body))
		}
	})

	t.Run("body为非bytes时报错", func(t *testing.T) {
		payload := map[string]any{
			"method": "POST",
			"path":   "/echo",
			"body":   "not-bytes",
		}

		encoded, err := poculum.DumpPoculum(payload)
		if err != nil {
			t.Fatalf("序列化payload失败: %v", err)
		}

		if _, err := BytesToRequest(encoded); err == nil {
			t.Fatalf("期望非bytes的body导致错误，但未返回错误")
		}
	})
}

func TestResponseToBytes(t *testing.T) {
	t.Run("正常序列化响应", func(t *testing.T) {
		resp := &Response{StatusCode: 201, Headers: map[string]string{"Content-Type": "bytes"}, Body: []byte("ok")}

		encoded, err := ResponseToBytes(resp)
		if err != nil {
			t.Fatalf("序列化响应失败: %v", err)
		}

		decoded, err := poculum.LoadPoculum(encoded)
		if err != nil {
			t.Fatalf("反序列化响应失败: %v", err)
		}

		m, ok := decoded.(map[string]any)
		if !ok {
			t.Fatalf("反序列化结果类型错误: %T", decoded)
		}

		if status, ok := m["status"].(uint32); !ok || status != 201 {
			t.Fatalf("状态码不匹配，得到 %v(%T)", m["status"], m["status"])
		}

		headers, ok := m["headers"].(map[string]any)
		if !ok {
			t.Fatalf("头部类型错误: %T", m["headers"])
		}
		expectedHeaders := map[string]any{"Content-Type": "bytes"}
		if !reflect.DeepEqual(headers, expectedHeaders) {
			t.Fatalf("头部不匹配，期望 %+v，实际 %+v", expectedHeaders, headers)
		}

		if body, ok := m["body"].([]byte); !ok || string(body) != "ok" {
			t.Fatalf("body 不匹配，得到 %v(%T)", m["body"], m["body"])
		}
	})

	t.Run("nil响应时报错", func(t *testing.T) {
		if _, err := ResponseToBytes(nil); err == nil {
			t.Fatalf("期望 nil 响应时报错，但未返回错误")
		}
	})
}
