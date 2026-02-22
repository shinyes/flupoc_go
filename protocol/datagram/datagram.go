// Package datagram 提供 Flupoc 协议的数据报序列化。
package datagram

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/cykyes/flupoc-go/protocol/head"
)

// Datagram 表示一个完整的 Flupoc 协议帧。
type Datagram struct {
	Head *head.Header
	Data []byte
}

// New 根据传入参数创建新的数据报。
func New(channelID uint16, msgType uint8, data []byte) *Datagram {
	return &Datagram{
		Head: &head.Header{
			Protocol:   head.ProtocolID,
			Type:       msgType,
			ChannelID:  channelID,
			DataLength: uint32(len(data)),
		},
		Data: data,
	}
}

// Serialize 将数据报编码成字节流。
func (d *Datagram) Serialize() []byte {
	if d.Head == nil {
		d.Head = &head.Header{Protocol: head.ProtocolID}
	}
	d.Head.DataLength = uint32(len(d.Data))

	buf := make([]byte, head.HeaderSize+len(d.Data))
	copy(buf[:head.HeaderSize], d.Head.Serialize())

	// 写数据
	if len(d.Data) > 0 {
		copy(buf[head.HeaderSize:], d.Data)
	}

	return buf
}

// Parse 从字节中解码数据报。
func Parse(data []byte) (*Datagram, error) {
	if len(data) < head.HeaderSize {
		slog.Error("数据报过短", "实际长度：", len(data), "标准长度：", head.HeaderSize)
		return nil, fmt.Errorf("数据报过短: %d 字节，至少需要 %d", len(data), head.HeaderSize)
	}

	h, err := head.Parse(data[:head.HeaderSize])
	if err != nil {
		return nil, err
	}

	expectedLen := head.HeaderSize + int(h.DataLength)
	if len(data) != expectedLen {
		slog.Error("负载数据部分长度与帧头中给出的长度不匹配", "实际长度：", len(data), "期望长度：", expectedLen)
		return nil, fmt.Errorf("数据报长度不匹配: 得到 %d，期望 %d", len(data), expectedLen)
	}

	var payload []byte
	if h.DataLength > 0 {
		payload = data[head.HeaderSize:]
	}

	return &Datagram{Head: h, Data: payload}, nil
}

// ReadFrom 从 io.Reader 中读取完整的数据报。
func ReadFrom(r io.Reader) (*Datagram, error) {
	h, err := head.ReadFrom(r)
	if err != nil {
		return nil, err
	}

	var data []byte
	if h.DataLength > 0 {
		data = make([]byte, h.DataLength)
		if _, err := io.ReadFull(r, data); err != nil {
			slog.Error("读取数据负载出现错误")
			return nil, fmt.Errorf("读取数据负载: %w", err)
		}
	}

	return &Datagram{Head: h, Data: data}, nil
}
