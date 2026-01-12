package main

import (
	"flag"
	"log"
	"strings"

	"github.com/cykyes/flupoc-go/protocol/service"
	"github.com/cykyes/flupoc-go/router"
	"github.com/cykyes/flupoc-go/transport"
)

// 使用示例:
// go run .\cmd\echo_server\main.go --addrs="127.0.0.1:5128" --cert="path/to/cert.crt" --key="path/to/key.key"

func main() {
	addrs := flag.String("addrs", "127.0.0.1:5128", "监听地址 (逗号分隔)")
	certFile := flag.String("cert", "", "TLS 证书文件路径")
	keyFile := flag.String("key", "", "TLS 私钥文件路径")
	flag.Parse()

	if *certFile == "" || *keyFile == "" {
		log.Fatal("必须提供 --cert 和 --key 参数")
	}

	parsedAddrs := parseAddrs(*addrs)
	if len(parsedAddrs) == 0 {
		log.Fatal("未提供有效地址")
	}

	// 创建路由器并注册回显路由
	r := router.NewRouter()
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		// 原样返回接收到的二进制数据
		log.Printf("收到请求，数据大小: %d 字节", len(ctx.RequestBody))
		return router.Bytes(ctx.RequestBody), nil
	})

	// 创建协议层服务并启动 TLS 服务器
	svc := service.New(r, service.Options{})
	log.Printf("回显服务器启动中，监听地址: %s", strings.Join(parsedAddrs, ", "))
	if err := transport.ListenAndServeTLS(transport.Config{
		Addrs:       parsedAddrs,
		CertFile:    *certFile,
		KeyFile:     *keyFile,
		ConnService: svc.Handle,
	}); err != nil {
		log.Fatalf("服务器退出: %v", err)
	}
}

func parseAddrs(s string) []string {
	var result []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			result = append(result, p)
		}
	}
	return result
}
