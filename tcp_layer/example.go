package tcplayer

import (
	"log"

	"github.com/cykyes/flupoc-go/protocol/service"
	"github.com/cykyes/flupoc-go/router"
)

// ExampleServer demonstrates starting a multi-address TLS server.
func ExampleServer() {
	// 1. 创建路由器
	r := router.NewRouter()
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		return router.Bytes(ctx.RequestBody), nil
	})

	// 2. 创建协议层连接处理器
	opts := service.DefaultHandlerOptions()
	connHandler, err := service.NewConnectionService(r, opts)
	if err != nil {
		log.Fatalf("创建连接处理器失败: %v", err)
	}

	// 3. 启动 TLS 服务器
	addrs := []string{"127.0.0.1:5128", "[::]:8443"}
	cert := "server.crt"
	key := "server.key"

	if err := ListenAndServeTLS(addrs, cert, key, connHandler.AsConnService()); err != nil {
		log.Fatalf("服务器错误: %v", err)
	}
}
