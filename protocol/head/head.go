// Package head 定义 Flupoc 协议帧头结构。
package head

import (
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
)

// 协议常量
const (
	ProtocolID uint8 = 0xCF // Flupoc 协议标识

	// 消息类型
	MsgPing          uint8 = 0x01 // 心跳请求
	MsgPong          uint8 = 0x02 // 心跳响应
	MsgRequest       uint8 = 0x03 // 请求消息
	MsgResponse      uint8 = 0x04 // 响应消息
	MsgCreateChannel uint8 = 0x05 // 打开持久通道
	MsgCloseChannel  uint8 = 0x06 // 关闭持久通道

	// HeaderSize 表示帧头固定大小（8 字节）
	HeaderSize = 8

	// MaxDataLength 限制最大数据长度（100MB）
	MaxDataLength = 100 << 20
)

// Header 表示 Flupoc 帧头结构。
type Header struct {
	Protocol   uint8  // 协议标识（必须是 ProtocolID）
	Type       uint8  // 消息类型
	ChannelID  uint16 // 通道编号
	DataLength uint32 // 数据载荷长度
}

// Validate 检查帧头是否有效。
func (h *Header) Validate() error {
	if h.Protocol != ProtocolID {
		slog.Error("检查协议帧头时发现非法协议号：", "协议号：", h.Protocol)
		return fmt.Errorf("invalid protocol id: 0x%02X, expected 0x%02X", h.Protocol, ProtocolID)
	}
	if h.DataLength > MaxDataLength {
		slog.Error("检查协议帧头时发现数据长度超出限制：", "数据长度：", h.DataLength)
		return fmt.Errorf("data length %d exceeds maximum %d", h.DataLength, MaxDataLength)
	}
	return nil
}

// Serialize 将帧头序列化为字节。
func (h *Header) Serialize() []byte {
	buf := make([]byte, HeaderSize)
	buf[0] = h.Protocol
	buf[1] = h.Type
	binary.BigEndian.PutUint16(buf[2:4], h.ChannelID)
	binary.BigEndian.PutUint32(buf[4:8], h.DataLength)
	return buf
}

// Parse 从字节解码帧头。
func Parse(data []byte) (*Header, error) {
	if len(data) < HeaderSize {
		slog.Error("解析协议帧头时发现数据过短：", "数据长度：", len(data))
		return nil, fmt.Errorf("header too short: %d bytes, need %d", len(data), HeaderSize)
	}

	h := &Header{
		Protocol:   data[0],
		Type:       data[1],
		ChannelID:  binary.BigEndian.Uint16(data[2:4]),
		DataLength: binary.BigEndian.Uint32(data[4:8]),
	}

	if err := h.Validate(); err != nil {
		return nil, err
	}

	return h, nil
}

// ReadFrom 从 io.Reader 中读取帧头。
func ReadFrom(r io.Reader) (*Header, error) {
	buf := make([]byte, HeaderSize)
	if _, err := io.ReadFull(r, buf); err != nil {
		return nil, err
	}
	return Parse(buf)
}
