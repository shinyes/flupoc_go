// Package service 提供 Flupoc 协议的连接处理逻辑。
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

// ConnService 是连接处理函数的类型。
type ConnService func(ctx context.Context, conn net.Conn) error

// Options 配置连接处理行为。
type Options struct {
	// IdleTimeout 空闲超时时间，超时后关闭连接。0 表示无超时。
	IdleTimeout time.Duration
	// PingInterval 心跳发送间隔。0 表示不发送心跳。
	PingInterval time.Duration
}

// Service 负责处理单个连接的完整生命周期。
type Service struct {
	router *router.Router
	opts   Options

	// 统计信息
	idleClosed uint64
}

// New 创建新的连接服务。router 不能为 nil。
func New(r *router.Router, opts Options) *Service {
	if r == nil {
		panic("service: router 不能为空")
	}
	// 规范化选项
	if opts.PingInterval < 0 {
		opts.PingInterval = 0
	}
	if opts.IdleTimeout < 0 {
		opts.IdleTimeout = 0
	}
	return &Service{router: r, opts: opts}
}

// Stats 返回统计信息。
type Stats struct {
	IdleClosed uint64
}

// Stats 返回当前统计信息。
func (s *Service) Stats() Stats {
	return Stats{IdleClosed: atomic.LoadUint64(&s.idleClosed)}
}

// Handle 处理一个连接的完整生命周期。
// 此方法会阻塞直到连接关闭或 ctx 被取消。
// 调用方负责在此方法返回后关闭连接。
// 此方法满足 ConnService 类型。
func (s *Service) Handle(ctx context.Context, conn net.Conn) error {
	stopPing := make(chan struct{})
	defer close(stopPing)

	if s.opts.PingInterval > 0 {
		go s.pingLoop(ctx, conn, stopPing)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if s.opts.IdleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.opts.IdleTimeout))
		}

		dg, err := datagram.ReadFrom(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				atomic.AddUint64(&s.idleClosed, 1)
				return fmt.Errorf("连接空闲超时: %w", err)
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return nil // 正常关闭
			}
			return fmt.Errorf("读取数据报: %w", err)
		}

		if respDG := s.handleDatagram(dg); respDG != nil {
			if _, err := conn.Write(respDG.Serialize()); err != nil {
				return fmt.Errorf("写入响应: %w", err)
			}
		}
	}
}

// pingLoop 定期发送心跳包。
func (s *Service) pingLoop(ctx context.Context, conn net.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(s.opts.PingInterval)
	defer ticker.Stop()

	pingData := datagram.New(0, head.MsgPing, nil).Serialize()

	for {
		select {
		case <-stop:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := conn.Write(pingData); err != nil {
				return
			}
		}
	}
}

// handleDatagram 根据消息类型处理数据报。
func (s *Service) handleDatagram(dg *datagram.Datagram) *datagram.Datagram {
	switch dg.Head.Type {
	case head.MsgPing:
		return datagram.New(dg.Head.ChannelID, head.MsgPong, nil)
	case head.MsgPong:
		return nil
	case head.MsgRequest:
		return s.handleRequest(dg)
	default:
		log.Printf("不支持的消息类型: %d", dg.Head.Type)
		return nil
	}
}

// handleRequest 处理请求类型的数据报。
func (s *Service) handleRequest(dg *datagram.Datagram) *datagram.Datagram {
	req, err := router.BytesToRequest(dg.Data)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, err)
	}

	resp, err := s.router.ServeRequest(req)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, err)
	}

	payload, err := router.ResponseToBytes(resp)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, err)
	}

	return datagram.New(dg.Head.ChannelID, head.MsgResponse, payload)
}

// buildErrorDatagram 构建错误响应数据报。
func buildErrorDatagram(channelID uint16, err error) *datagram.Datagram {
	resp := router.Error(500, err.Error())
	payload, _ := router.ResponseToBytes(resp) // Error 响应编码不会失败
	return datagram.New(channelID, head.MsgResponse, payload)
}
