// Package tcplayer 提供 TLS TCP 服务器，负责创建和管理 TLS 连接。
// 连接的协议处理由 protocol/handler 包负责。
package tcplayer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cykyes/flupoc-go/protocol/service"
)

// ConnService 定义连接处理函数类型。
// 由协议层提供具体实现。
type ConnService = service.ConnService

// Server 表示一个 TLS TCP 服务器。
// 只负责创建和接受 TLS 连接，具体的协议处理交给 ConnService。
type Server struct {
	addr        string
	tlsConfig   *tls.Config
	listener    net.Listener
	connService ConnService

	activeConns int64
}

// ServerStats 包含服务器统计信息。
type ServerStats struct {
	ActiveConns int64
}

// Addr 返回服务器的监听地址。
func (s *Server) Addr() string {
	return s.addr
}

// NewServer 创建 TLS TCP 服务器。
// connService 由协议层提供，负责处理连接的生命周期。
func NewServer(addr, certFile, keyFile string, connService ConnService) (*Server, error) {
	if connService == nil {
		return nil, errors.New("连接处理函数不能为空")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载 TLS 密钥对: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return &Server{
		addr:        addr,
		tlsConfig:   cfg,
		connService: connService,
	}, nil
}

// ServeTLS 在多个地址上启动 TLS 服务器并返回实例。
// 调用方需持有 ctx，ctx 取消后所有监听器会退出。
func ServeTLS(ctx context.Context, addrs []string, certFile, keyFile string, connService ConnService) ([]*Server, error) {
	if len(addrs) == 0 {
		return nil, errors.New("地址列表为空")
	}

	servers := make([]*Server, 0, len(addrs))
	for _, addr := range addrs {
		srv, err := NewServer(addr, certFile, keyFile, connService)
		if err != nil {
			return nil, fmt.Errorf("创建服务器 (%s): %w", addr, err)
		}
		servers = append(servers, srv)
	}

	for _, srv := range servers {
		s := srv
		go func() {
			if err := s.Start(ctx); err != nil {
				log.Printf("服务器启动失败 (%s): %v", s.Addr(), err)
			}
		}()
	}

	return servers, nil
}

// ListenAndServeTLS 启动服务器并阻塞直到被中断。
func ListenAndServeTLS(addrs []string, certFile, keyFile string, connService ConnService) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	servers, err := ServeTLS(ctx, addrs, certFile, keyFile, connService)
	if err != nil {
		return err
	}

	log.Printf("服务器已启动，监听: %s，按 Ctrl+C 退出", strings.Join(addrs, ", "))
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	<-sigCh

	cancel()
	var closeErr error
	for _, srv := range servers {
		if err := srv.Close(); err != nil {
			log.Printf("关闭服务器错误 (%s): %v", srv.Addr(), err)
			if closeErr == nil {
				closeErr = err
			}
		}
	}
	return closeErr
}

// Start 开始接收 TLS 连接。
func (s *Server) Start(ctx context.Context) error {
	if s.listener != nil {
		return errors.New("服务器已启动")
	}

	ln, err := tls.Listen("tcp", s.addr, s.tlsConfig)
	if err != nil {
		return fmt.Errorf("监听 %s: %w", s.addr, err)
	}
	s.listener = ln

	// Close listener when context is canceled.
	go func() {
		<-ctx.Done()
		_ = ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			// 监听器关闭
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			// 对于超时错误，短暂等待后重试
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				log.Printf("accept 超时: %v", err)
				time.Sleep(time.Millisecond * 100)
				continue
			}
			return fmt.Errorf("接受连接: %w", err)
		}

		atomic.AddInt64(&s.activeConns, 1)
		go s.handleConn(ctx, conn)
	}
}

// Close 停止接收新连接。
func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

// handleConn 处理单个连接，将控制权交给协议层的 connHandler。
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		atomic.AddInt64(&s.activeConns, -1)
		conn.Close()
	}()

	// 将连接交给协议层处理
	if err := s.connService(ctx, conn); err != nil {
		log.Printf("连接处理结束: %v", err)
	}
}

// Stats 返回服务器统计信息。
func (s *Server) Stats() ServerStats {
	return ServerStats{
		ActiveConns: atomic.LoadInt64(&s.activeConns),
	}
}
