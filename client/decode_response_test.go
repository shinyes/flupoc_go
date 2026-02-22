package client

import (
	"testing"

	"github.com/cykyes/flupoc-go/poculum"
)

func TestDecodeResponseRequiresStatus(t *testing.T) {
	payload, err := poculum.DumpPoculum(map[string]any{
		"headers": map[string]any{"k": "v"},
		"body":    []byte("ok"),
	})
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	_, err = decodeResponse(payload)
	if err == nil {
		t.Fatalf("期望缺少 status 时返回错误，但得到 nil")
	}
}

func TestDecodeResponseRejectsInvalidStatus(t *testing.T) {
	payload, err := poculum.DumpPoculum(map[string]any{
		"status": 0,
		"body":   []byte("ok"),
	})
	if err != nil {
		t.Fatalf("编码失败: %v", err)
	}

	_, err = decodeResponse(payload)
	if err == nil {
		t.Fatalf("期望非法 status 返回错误，但得到 nil")
	}
}
