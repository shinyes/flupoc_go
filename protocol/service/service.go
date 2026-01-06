// Package handler 提供 Flupoc 协议的连接处理逻辑。
// 此包负责管理连接生命周期、数据报读写和请求分发。
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync/atomic"
	"time"

	"github.com/cykyes/flupoc-go/protocol/datagram"
	"github.com/cykyes/flupoc-go/protocol/head"
	"github.com/cykyes/flupoc-go/router"
)

// ConnectionService 负责处理单个连接的完整生命周期。
type ConnectionService struct {
	router *router.Router
	opts   ServiceOptions

	// 统计信息
	idleClosed uint64
}

// ServiceOptions 配置连接处理行为。
type ServiceOptions struct {
	// IdleTimeout 空闲超时时间，超时后关闭连接。0 表示无超时。
	IdleTimeout time.Duration
	// PingInterval 心跳发送间隔。0 表示不发送心跳。
	PingInterval time.Duration
}

// DefaultHandlerOptions 返回默认的处理器选项。
func DefaultHandlerOptions() ServiceOptions {
	return ServiceOptions{
		IdleTimeout:  0,
		PingInterval: 0,
	}
}

// Normalize 规范化选项值。
func (o ServiceOptions) Normalize() ServiceOptions {
	if o.PingInterval < 0 {
		o.PingInterval = 0
	}
	if o.IdleTimeout < 0 {
		o.IdleTimeout = 0
	}
	return o
}

// NewConnectionService 创建新的连接处理器。
func NewConnectionService(r *router.Router, opts ServiceOptions) (*ConnectionService, error) {
	if r == nil {
		return nil, errors.New("路由器不能为空")
	}
	return &ConnectionService{
		router: r,
		opts:   opts.Normalize(),
	}, nil
}

// Stats 返回处理器统计信息。
type Stats struct {
	IdleClosed uint64
}

// Stats 返回当前统计信息。
func (h *ConnectionService) Stats() Stats {
	return Stats{
		IdleClosed: atomic.LoadUint64(&h.idleClosed),
	}
}

// HandleConnection 处理一个连接的完整生命周期。
// 此方法会阻塞直到连接关闭或 ctx 被取消。
// 调用方负责在此方法返回后关闭连接。
func (h *ConnectionService) HandleConnection(ctx context.Context, conn net.Conn) error {
	stopPing := make(chan struct{})
	defer close(stopPing)

	if h.opts.PingInterval > 0 {
		go h.pingLoop(ctx, conn, stopPing)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if h.opts.IdleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(h.opts.IdleTimeout))
		}

		dg, err := datagram.ReadFrom(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				atomic.AddUint64(&h.idleClosed, 1)
				return fmt.Errorf("连接空闲超时: %w", err)
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil // 正常关闭
			}
			return fmt.Errorf("读取数据报: %w", err)
		}

		respDG, err := h.handleDatagram(dg)
		if err != nil {
			log.Printf("处理数据报: %v", err)
			continue
		}
		if respDG == nil {
			continue
		}

		raw, err := respDG.Serialize()
		if err != nil {
			log.Printf("序列化响应: %v", err)
			continue
		}

		if _, err := conn.Write(raw); err != nil {
			return fmt.Errorf("写入响应: %w", err)
		}
	}
}

// pingLoop 定期发送心跳包。
func (h *ConnectionService) pingLoop(ctx context.Context, conn net.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(h.opts.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			dg := datagram.New(0, head.MsgPing, nil)
			raw, err := dg.Serialize()
			if err != nil {
				log.Printf("序列化 ping: %v", err)
				continue
			}
			if _, err := conn.Write(raw); err != nil {
				log.Printf("发送 ping: %v", err)
				return
			}
		}
	}
}

// handleDatagram 根据消息类型处理数据报。
func (h *ConnectionService) handleDatagram(dg *datagram.Datagram) (*datagram.Datagram, error) {
	switch dg.Head.Type {
	case head.MsgPing:
		return datagram.New(dg.Head.ChannelID, head.MsgPong, nil), nil
	case head.MsgPong:
		return nil, nil
	case head.MsgRequest:
		return h.handleRequest(dg)
	default:
		return nil, fmt.Errorf("不支持的消息类型: %d", dg.Head.Type)
	}
}

// handleRequest 处理请求类型的数据报。
func (h *ConnectionService) handleRequest(dg *datagram.Datagram) (*datagram.Datagram, error) {
	req, err := router.BytesToRequest(dg.Data)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, fmt.Errorf("解码请求: %w", err))
	}

	resp, err := h.router.ServeRequest(req)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, err)
	}

	payload, err := router.ResponseToBytes(resp)
	if err != nil {
		return nil, fmt.Errorf("编码响应: %w", err)
	}

	return datagram.New(dg.Head.ChannelID, head.MsgResponse, payload), nil
}

// buildErrorDatagram 构建错误响应数据报。
func buildErrorDatagram(channelID uint16, err error) (*datagram.Datagram, error) {
	resp := router.Error(500, err.Error())
	payload, encErr := router.ResponseToBytes(resp)
	if encErr != nil {
		return nil, fmt.Errorf("编码错误响应: %w", encErr)
	}
	return datagram.New(channelID, head.MsgResponse, payload), nil
}

// ConnService 是连接处理函数的类型。
// 用于 tcp_layer 调用协议层处理连接。
type ConnService func(ctx context.Context, conn net.Conn) error

// AsConnService 将 ConnectionService 转换为 ConnService 函数。
func (h *ConnectionService) AsConnService() ConnService {
	return h.HandleConnection
}
