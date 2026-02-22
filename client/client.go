// Package client 提供 Flupoc 协议的 TLS 客户端。
package client

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/cykyes/flupoc-go/poculum"
	"github.com/cykyes/flupoc-go/protocol/datagram"
	"github.com/cykyes/flupoc-go/protocol/head"
	"github.com/cykyes/flupoc-go/router"
)

// Options 配置客户端行为。
type Options struct {
	// TLS 设置
	CertFile   string // 客户端证书路径（可选，支持 mTLS）
	KeyFile    string // 客户端私钥路径（可选，支持 mTLS）
	CAFile     string // 用于校验服务器的 CA 证书
	Insecure   bool   // 是否跳过服务器证书校验
	ServerName string // 覆写 TLS 验证使用的主机名

	// 超时设置
	DialTimeout  time.Duration // 建连超时（默认 5s）
	ReadTimeout  time.Duration // 读响应超时（默认 5s）
	WriteTimeout time.Duration // 写请求超时（默认 5s）
}

// Client 表示可复用的 Flupoc 协议客户端。
type Client struct {
	tlsConf *tls.Config
	opts    Options
}

// New 根据配置创建客户端。
func New(opts Options) (*Client, error) {
	opts = normalizeOptions(opts)

	conf, err := buildTLSConfig(opts)
	if err != nil {
		return nil, err
	}

	return &Client{tlsConf: conf, opts: opts}, nil
}

func normalizeOptions(opts Options) Options {
	if opts.DialTimeout <= 0 {
		opts.DialTimeout = 5 * time.Second
	}
	if opts.ReadTimeout <= 0 {
		opts.ReadTimeout = 5 * time.Second
	}
	if opts.WriteTimeout <= 0 {
		opts.WriteTimeout = 5 * time.Second
	}
	return opts
}

// Do 发送请求并返回响应。
func (c *Client) Do(addr, method, path string, body []byte) (*router.Response, error) {
	d := &net.Dialer{Timeout: c.opts.DialTimeout}
	conn, err := tls.DialWithDialer(d, "tcp", addr, c.tlsConf)
	if err != nil {
		return nil, fmt.Errorf("连接: %w", err)
	}
	defer conn.Close()

	payload, err := poculum.DumpPoculum(map[string]any{
		"method": method,
		"path":   path,
		"body":   body,
	})
	if err != nil {
		return nil, fmt.Errorf("编码请求: %w", err)
	}

	dg := datagram.New(1, head.MsgRequest, payload)

	if err := conn.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout)); err != nil {
		return nil, err
	}
	if _, err := dg.WriteTo(conn); err != nil {
		return nil, fmt.Errorf("发送: %w", err)
	}

	for {
		// 每次读取前刷新超时，使得 PING/PONG 往返也能保持连接活跃
		if err := conn.SetReadDeadline(time.Now().Add(c.opts.ReadTimeout)); err != nil {
			return nil, err
		}

		dgResp, err := datagram.ReadFrom(conn)
		if err != nil {
			return nil, fmt.Errorf("读取响应: %w", err)
		}

		switch dgResp.Head.Type {
		case head.MsgPing:
			// 服务器心跳，立即回复 PONG 继续等待真实响应
			pong := datagram.New(dgResp.Head.ChannelID, head.MsgPong, nil)
			if err := conn.SetWriteDeadline(time.Now().Add(c.opts.WriteTimeout)); err != nil {
				return nil, err
			}
			if _, err := pong.WriteTo(conn); err != nil {
				return nil, fmt.Errorf("发送 PONG: %w", err)
			}
			continue

		case head.MsgResponse:
			return decodeResponse(dgResp.Data)

		default:
			return nil, fmt.Errorf("意外的消息类型: %d", dgResp.Head.Type)
		}
	}
}

// Get 发送 GET 请求。
func (c *Client) Get(addr, path string) (*router.Response, error) {
	return c.Do(addr, "GET", path, nil)
}

// Post 发送 POST 请求。
func (c *Client) Post(addr, path string, body []byte) (*router.Response, error) {
	return c.Do(addr, "POST", path, body)
}

// Put 发送 PUT 请求。
func (c *Client) Put(addr, path string, body []byte) (*router.Response, error) {
	return c.Do(addr, "PUT", path, body)
}

// Delete 发送 DELETE 请求。
func (c *Client) Delete(addr, path string) (*router.Response, error) {
	return c.Do(addr, "DELETE", path, nil)
}

// Call 是一次性请求助手，每次调用都会创建新客户端。
func Call(addr string, opts Options, method, path string, body []byte) (*router.Response, error) {
	cli, err := New(opts)
	if err != nil {
		return nil, err
	}
	return cli.Do(addr, method, path, body)
}

func buildTLSConfig(opt Options) (*tls.Config, error) {
	conf := &tls.Config{InsecureSkipVerify: opt.Insecure}

	if opt.CertFile != "" && opt.KeyFile != "" {
		cert, err := tls.LoadX509KeyPair(opt.CertFile, opt.KeyFile)
		if err != nil {
			return nil, fmt.Errorf("加载客户端证书: %w", err)
		}
		conf.Certificates = []tls.Certificate{cert}
	}

	if opt.CAFile != "" {
		caPEM, err := os.ReadFile(opt.CAFile)
		if err != nil {
			return nil, fmt.Errorf("读取 CA 证书: %w", err)
		}
		pool := x509.NewCertPool()
		if ok := pool.AppendCertsFromPEM(caPEM); !ok {
			return nil, fmt.Errorf("解析 CA 证书失败")
		}
		conf.RootCAs = pool
	}

	if opt.ServerName != "" {
		conf.ServerName = opt.ServerName
	}

	return conf, nil
}

func decodeResponse(data []byte) (*router.Response, error) {
	decoded, err := poculum.LoadPoculum(data)
	if err != nil {
		return nil, fmt.Errorf("解码响应: %w", err)
	}

	m, ok := decoded.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("响应结构错误: %T", decoded)
	}

	resp := &router.Response{}

	statusRaw, ok := m["status"]
	if !ok {
		return nil, fmt.Errorf("响应缺少 status 字段")
	}
	resp.StatusCode = toInt(statusRaw)
	if resp.StatusCode <= 0 {
		return nil, fmt.Errorf("响应 status 非法: %v", statusRaw)
	}

	if v, ok := m["headers"].(map[string]any); ok {
		resp.Headers = make(map[string]string, len(v))
		for k, val := range v {
			if s, ok := val.(string); ok {
				resp.Headers[k] = s
			}
		}
	}

	if body, ok := m["body"].([]byte); ok {
		resp.Body = body
	}
	return resp, nil
}

func toInt(v any) int {
	switch n := v.(type) {
	case uint32:
		return int(n)
	case uint64:
		return int(n)
	case int64:
		return int(n)
	case int:
		return n
	case float64:
		return int(n)
	default:
		return 0
	}
}
