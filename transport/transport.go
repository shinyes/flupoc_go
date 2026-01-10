// Package transport 提供 TLS TCP 服务器，负责创建和管理 TLS 连接。
package transport

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

	"github.com/cykyes/flupoc-go/protocol/service"
)

// Server 表示一个 TLS TCP 服务器。
type Server struct {
	addr        string
	tlsConfig   *tls.Config
	listener    net.Listener
	connService service.ConnService
	activeConns atomic.Int64
}

// Config 服务器配置。
type Config struct {
	Addrs       []string
	CertFile    string
	KeyFile     string
	ConnService service.ConnService
}

// ServeTLS 启动 TLS 服务器并返回实例，ctx 取消时服务器关闭。
func ServeTLS(ctx context.Context, cfg Config) ([]*Server, error) {
	if len(cfg.Addrs) == 0 {
		return nil, errors.New("地址列表为空")
	}
	if cfg.ConnService == nil {
		return nil, errors.New("连接处理函数不能为空")
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("加载 TLS 密钥对: %w", err)
	}
	tlsCfg := &tls.Config{Certificates: []tls.Certificate{cert}, MinVersion: tls.VersionTLS12}

	servers := make([]*Server, 0, len(cfg.Addrs))
	for _, addr := range cfg.Addrs {
		srv := &Server{addr: addr, tlsConfig: tlsCfg, connService: cfg.ConnService}
		servers = append(servers, srv)
		go srv.serve(ctx)
	}
	return servers, nil
}

// ListenAndServeTLS 启动服务器并阻塞直到收到中断信号。
func ListenAndServeTLS(cfg Config) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	servers, err := ServeTLS(ctx, cfg)
	if err != nil {
		return err
	}

	log.Printf("服务器已启动，监听: %s", strings.Join(cfg.Addrs, ", "))
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	<-sig

	cancel()
	var closeErr error
	for _, srv := range servers {
		if err := srv.Close(); err != nil {
			closeErr = err
		}
	}
	return closeErr
}

// serve 监听并处理传入连接。
func (s *Server) serve(ctx context.Context) {
	ln, err := tls.Listen("tcp", s.addr, s.tlsConfig)
	if err != nil {
		log.Printf("监听失败 (%s): %v", s.addr, err)
		return
	}
	s.listener = ln

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			if !errors.Is(err, net.ErrClosed) {
				log.Printf("接受连接失败 (%s): %v", s.addr, err)
			}
			return
		}
		s.activeConns.Add(1)
		go s.handleConn(ctx, conn)
	}
}

// handleConn 处理单个连接。
func (s *Server) handleConn(ctx context.Context, conn net.Conn) {
	defer func() {
		s.activeConns.Add(-1)
		conn.Close()
	}()
	if err := s.connService(ctx, conn); err != nil {
		log.Printf("连接处理结束: %v", err)
	}
}

// Close 停止接收新连接。
func (s *Server) Close() error {
	if s.listener == nil {
		return nil
	}
	return s.listener.Close()
}

// Addr 返回监听地址。
func (s *Server) Addr() string { return s.addr }

// ActiveConns 返回当前活跃连接数。
func (s *Server) ActiveConns() int64 { return s.activeConns.Load() }
