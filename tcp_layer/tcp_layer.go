// Package tcplayer 提供基于 Flupoc 协议的 TLS TCP 服务器。
package tcplayer

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/cykyes/flupoc-go/protocol/datagram"
	"github.com/cykyes/flupoc-go/protocol/head"
	"github.com/cykyes/flupoc-go/router"
)

type Server struct {
	addr      string
	router    *router.Router
	tlsConfig *tls.Config
	listener  net.Listener

	opts ServerOptions

	activeConns int64
	idleClosed  uint64
}

type ServerOptions struct {
	IdleTimeout  time.Duration
	PingInterval time.Duration
}

type ServerStats struct {
	ActiveConns int64
	IdleClosed  uint64
}

func defaultServerOptions() ServerOptions {
	return ServerOptions{
		IdleTimeout:  0,
		PingInterval: 0,
	}
}

func (o ServerOptions) normalize() ServerOptions {
	if o.PingInterval < 0 {
		o.PingInterval = 0
	}
	if o.IdleTimeout < 0 {
		o.IdleTimeout = 0
	}
	return o
}

// Addr 返回服务器的监听地址。
func (s *Server) Addr() string {
	return s.addr
}

// NewServer 创建用于路由数据报的 TLS TCP 服务器。
func NewServer(addr, certFile, keyFile string, r *router.Router, opts ServerOptions) (*Server, error) {
	if r == nil {
		return nil, errors.New("路由器不能为空")
	}
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("加载 TLS 密钥对: %w", err)
	}

	cfg := &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS12,
	}

	return &Server{addr: addr, router: r, tlsConfig: cfg, opts: opts}, nil
}

// ServeTLS 在多个地址上启动 TLS 服务器并返回实例。
// 调用方需持有 ctx，ctx 取消后所有监听器会退出。
func ServeTLS(ctx context.Context, addrs []string, certFile, keyFile string, r *router.Router, opts *ServerOptions) ([]*Server, error) {
	if len(addrs) == 0 {
		return nil, errors.New("地址列表为空")
	}

	finalOpts := defaultServerOptions()
	if opts != nil {
		finalOpts = opts.normalize()
	}

	servers := make([]*Server, 0, len(addrs))
	for _, addr := range addrs {
		srv, err := NewServer(addr, certFile, keyFile, r, finalOpts)
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
func ListenAndServeTLS(addrs []string, certFile, keyFile string, r *router.Router, opts *ServerOptions) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	servers, err := ServeTLS(ctx, addrs, certFile, keyFile, r, opts)
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
			// 监听器关闭或临时错误。
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			if ne, ok := err.(net.Error); ok && ne.Temporary() {
				log.Printf("临时 accept 错误: %v", err)
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

func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		atomic.AddInt64(&s.activeConns, -1)
		conn.Close()
	}()

	stopPing := make(chan struct{})
	defer close(stopPing)
	if s.opts.PingInterval > 0 {
		go s.pingLoop(ctx, conn, stopPing)
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if s.opts.IdleTimeout > 0 {
			_ = conn.SetReadDeadline(time.Now().Add(s.opts.IdleTimeout))
		}

		dg, err := datagram.ReadFrom(conn)
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				atomic.AddUint64(&s.idleClosed, 1)
			}
			if errors.Is(err, io.EOF) || errors.Is(err, net.ErrClosed) {
				return
			}
			log.Printf("读取数据报: %v", err)
			return
		}

		respDG, err := s.handleDatagram(dg)
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
			log.Printf("写入响应: %v", err)
			return
		}
	}
}

func (s *Server) pingLoop(ctx context.Context, conn net.Conn, stop <-chan struct{}) {
	ticker := time.NewTicker(s.opts.PingInterval)
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

// Stats returns current server connection statistics.
func (s *Server) Stats() ServerStats {
	return ServerStats{
		ActiveConns: atomic.LoadInt64(&s.activeConns),
		IdleClosed:  atomic.LoadUint64(&s.idleClosed),
	}
}

func (s *Server) handleDatagram(dg *datagram.Datagram) (*datagram.Datagram, error) {
	switch dg.Head.Type {
	case head.MsgPing:
		return datagram.New(dg.Head.ChannelID, head.MsgPong, nil), nil
	case head.MsgPong:
		return nil, nil
	case head.MsgRequest:
		return s.handleRequest(dg)
	default:
		return nil, fmt.Errorf("不支持的消息类型: %d", dg.Head.Type)
	}
}

func (s *Server) handleRequest(dg *datagram.Datagram) (*datagram.Datagram, error) {
	req, err := router.BytesToRequest(dg.Data)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, fmt.Errorf("解码请求: %w", err))
	}

	resp, err := s.router.ServeRequest(req)
	if err != nil {
		return buildErrorDatagram(dg.Head.ChannelID, err)
	}

	payload, err := router.ResponseToBytes(resp)
	if err != nil {
		return nil, fmt.Errorf("编码响应: %w", err)
	}

	return datagram.New(dg.Head.ChannelID, head.MsgResponse, payload), nil
}

func buildErrorDatagram(channelID uint16, err error) (*datagram.Datagram, error) {
	resp := router.Error(500, err.Error())
	payload, encErr := router.ResponseToBytes(resp)
	if encErr != nil {
		return nil, fmt.Errorf("编码错误响应: %w", encErr)
	}
	return datagram.New(channelID, head.MsgResponse, payload), nil
}
