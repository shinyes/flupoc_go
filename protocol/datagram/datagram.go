// Package datagram 提供 Flupoc 协议的数据报序列化。
package datagram

import (
	"encoding/binary"
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
	if d.Head.Protocol == 0 {
		d.Head.Protocol = head.ProtocolID
	}
	d.Head.DataLength = uint32(len(d.Data))

	buf := make([]byte, head.HeaderSize+len(d.Data))
	buf[0] = d.Head.Protocol
	buf[1] = d.Head.Type
	binary.BigEndian.PutUint16(buf[2:4], d.Head.ChannelID)
	binary.BigEndian.PutUint32(buf[4:8], d.Head.DataLength)

	// 写数据
	if len(d.Data) > 0 {
		copy(buf[head.HeaderSize:], d.Data)
	}

	return buf
}

// WriteTo 将数据报直接写入 io.Writer，避免拼接整帧产生额外分配。
func (d *Datagram) WriteTo(w io.Writer) (int64, error) {
	var protocol uint8 = head.ProtocolID
	var msgType uint8
	var channelID uint16
	if d.Head != nil {
		msgType = d.Head.Type
		channelID = d.Head.ChannelID
		if d.Head.Protocol != 0 {
			protocol = d.Head.Protocol
		}
	}
	dataLength := uint32(len(d.Data))

	var hdr [head.HeaderSize]byte
	hdr[0] = protocol
	hdr[1] = msgType
	binary.BigEndian.PutUint16(hdr[2:4], channelID)
	binary.BigEndian.PutUint32(hdr[4:8], dataLength)

	if len(d.Data) == 0 {
		n, err := writeAll(w, hdr[:])
		return int64(n), err
	}

	n1, err := writeAll(w, hdr[:])
	if err != nil {
		return int64(n1), err
	}
	n2, err := writeAll(w, d.Data)
	if err != nil {
		return int64(n1 + n2), err
	}
	return int64(n1 + n2), nil
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

func writeAll(w io.Writer, data []byte) (int, error) {
	total := 0
	for len(data) > 0 {
		n, err := w.Write(data)
		total += n
		if err != nil {
			return total, err
		}
		if n <= 0 {
			return total, io.ErrShortWrite
		}
		data = data[n:]
	}
	return total, nil
}
