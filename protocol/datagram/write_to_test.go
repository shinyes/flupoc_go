package datagram

import (
	"bytes"
	"io"
	"testing"

	"github.com/cykyes/flupoc-go/protocol/head"
)

type shortWriter struct {
	limit int
	buf   bytes.Buffer
}

func (w *shortWriter) Write(p []byte) (int, error) {
	if len(p) == 0 {
		return 0, nil
	}
	n := w.limit
	if n > len(p) {
		n = len(p)
	}
	return w.buf.Write(p[:n])
}

type zeroWriter struct{}

func (zeroWriter) Write([]byte) (int, error) { return 0, nil }

func TestWriteToRoundTrip(t *testing.T) {
	dg := New(9, head.MsgRequest, []byte("payload"))

	var out bytes.Buffer
	n, err := dg.WriteTo(&out)
	if err != nil {
		t.Fatalf("WriteTo 失败: %v", err)
	}
	if want := int64(head.HeaderSize + len("payload")); n != want {
		t.Fatalf("期望写入 %d 字节，实际 %d", want, n)
	}

	parsed, err := Parse(out.Bytes())
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if parsed.Head.ChannelID != 9 {
		t.Fatalf("期望 ChannelID=9，实际=%d", parsed.Head.ChannelID)
	}
	if string(parsed.Data) != "payload" {
		t.Fatalf("期望 payload，实际=%q", string(parsed.Data))
	}
}

func TestWriteToHandlesShortWrite(t *testing.T) {
	dg := New(1, head.MsgPing, []byte("abcde"))

	w := &shortWriter{limit: 2}
	n, err := dg.WriteTo(w)
	if err != nil {
		t.Fatalf("WriteTo 短写场景失败: %v", err)
	}
	if want := int64(head.HeaderSize + len("abcde")); n != want {
		t.Fatalf("期望写入 %d 字节，实际 %d", want, n)
	}

	got := w.buf.Bytes()
	parsed, err := Parse(got)
	if err != nil {
		t.Fatalf("解析写入数据失败: %v", err)
	}
	if string(parsed.Data) != "abcde" {
		t.Fatalf("期望 abcde，实际=%q", string(parsed.Data))
	}
}

func TestWriteToReturnsErrShortWriteOnZeroProgress(t *testing.T) {
	dg := New(1, head.MsgPing, []byte("x"))

	_, err := dg.WriteTo(zeroWriter{})
	if err == nil {
		t.Fatalf("期望返回错误，但得到 nil")
	}
	if err != io.ErrShortWrite {
		t.Fatalf("期望 io.ErrShortWrite，实际=%v", err)
	}
}
