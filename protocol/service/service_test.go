package service

import (
	"context"
	"io"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/cykyes/flupoc-go/protocol/head"
	"github.com/cykyes/flupoc-go/router"
)

type timeoutErr struct{}

func (timeoutErr) Error() string   { return "i/o timeout" }
func (timeoutErr) Timeout() bool   { return true }
func (timeoutErr) Temporary() bool { return true }

type dummyAddr string

func (a dummyAddr) Network() string { return "tcp" }
func (a dummyAddr) String() string  { return string(a) }

type timeoutAfterHeaderConn struct {
	header []byte
	stage  int
}

func (c *timeoutAfterHeaderConn) Read(p []byte) (int, error) {
	switch c.stage {
	case 0:
		n := copy(p, c.header)
		c.header = c.header[n:]
		if len(c.header) == 0 {
			c.stage = 1
		}
		return n, nil
	case 1:
		c.stage = 2
		return 0, timeoutErr{}
	default:
		return 0, io.EOF
	}
}

func (c *timeoutAfterHeaderConn) Write(p []byte) (int, error) { return len(p), nil }
func (c *timeoutAfterHeaderConn) Close() error                { return nil }
func (c *timeoutAfterHeaderConn) LocalAddr() net.Addr         { return dummyAddr("local") }
func (c *timeoutAfterHeaderConn) RemoteAddr() net.Addr        { return dummyAddr("remote") }
func (c *timeoutAfterHeaderConn) SetDeadline(time.Time) error { return nil }
func (c *timeoutAfterHeaderConn) SetReadDeadline(time.Time) error {
	return nil
}
func (c *timeoutAfterHeaderConn) SetWriteDeadline(time.Time) error {
	return nil
}

func TestHandleRecognizesWrappedTimeoutAsIdleTimeout(t *testing.T) {
	svc := New(router.NewRouter(), Options{IdleTimeout: time.Second})
	h := &head.Header{
		Protocol:   head.ProtocolID,
		Type:       head.MsgRequest,
		ChannelID:  1,
		DataLength: 1, // 触发读取 payload 分支，制造被包装的 timeout 错误
	}
	conn := &timeoutAfterHeaderConn{header: h.Serialize()}

	err := svc.Handle(context.Background(), conn)
	if err == nil {
		t.Fatalf("期望返回空闲超时错误，但得到 nil")
	}
	if !strings.Contains(err.Error(), "连接空闲超时") {
		t.Fatalf("期望错误包含连接空闲超时，实际: %v", err)
	}
	if got := svc.Stats().IdleClosed; got != 1 {
		t.Fatalf("期望 IdleClosed=1，实际 %d", got)
	}
}
