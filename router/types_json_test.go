package router

import "testing"

func TestJSONSuccess(t *testing.T) {
	resp := JSON(map[string]string{"name": "flupoc"})

	if resp.StatusCode != 200 {
		t.Fatalf("期望状态码 200，实际 %d", resp.StatusCode)
	}
	if got := resp.Headers["Content-Type"]; got != "application/json" {
		t.Fatalf("期望 Content-Type=application/json，实际 %s", got)
	}
	if string(resp.Body) != `{"name":"flupoc"}` {
		t.Fatalf("期望 JSON body 为 {\"name\":\"flupoc\"}，实际 %s", string(resp.Body))
	}
}

func TestJSONMarshalFailureReturns500(t *testing.T) {
	// channel 无法被 encoding/json 序列化
	resp := JSON(map[string]any{"bad": make(chan int)})

	if resp.StatusCode != 500 {
		t.Fatalf("期望状态码 500，实际 %d", resp.StatusCode)
	}
	if got := resp.Headers["Content-Type"]; got != "application/json" {
		t.Fatalf("期望 Content-Type=application/json，实际 %s", got)
	}
	if string(resp.Body) != `{"error":"json marshal failed"}` {
		t.Fatalf("期望错误 body 固定文案，实际 %s", string(resp.Body))
	}
}
