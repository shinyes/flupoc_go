package main

import (
	"flag"
	"log"
	"strings"
	"time"

	"github.com/cykyes/flupoc-go/router"
	tcplayer "github.com/cykyes/flupoc-go/tcp_layer"
)

// 这是一个带有高级配置的服务器示例
// 演示了如何配置心跳保活 (Ping/Pong) 和空闲超时 (IdleTimeout)

// 运行示例:
// go run ./cmd/advanced_server --cert="server.crt" --key="server.key"

func main() {
	addrs := flag.String("addrs", "127.0.0.1:5128", "监听地址 (逗号分隔)")
	certFile := flag.String("cert", "", "TLS 证书文件路径")
	keyFile := flag.String("key", "", "TLS 私钥文件路径")

	// 配置参数
	idleTimeout := flag.Duration("idle", 2*time.Minute, "空闲超时时间")
	pingInterval := flag.Duration("ping", 30*time.Second, "心跳发送间隔")

	flag.Parse()

	if *certFile == "" || *keyFile == "" {
		log.Fatal("错误: 必须提供 --cert 和 --key 参数")
	}

	// 1. 创建路由器
	r := router.NewRouter()

	// 添加一个简单的回显路由
	r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
		log.Printf("[路由] 收到 /echo 请求，数据长度: %d", len(ctx.RequestBody))
		return router.Bytes(ctx.RequestBody), nil
	})

	// 添加一个模拟长任务的路由，演示心跳在任务执行期间的作用
	// 注意：虽然业务处理时不会阻塞底层的读循环（因为是在 handleRequest 中处理），
	// 但客户端在等待响应时，心跳能确保连接不被中间设备切断。
	r.Get("/long-task", func(ctx *router.Context) (*router.Response, error) {
		log.Println("[路由] 开始执行长任务 (5秒)...")
		time.Sleep(5 * time.Second)
		log.Println("[路由] 长任务完成")
		return router.Text("任务完成"), nil
	})

	// 2. 配置服务器选项
	opts := &tcplayer.ServerOptions{
		// 如果连接在 2 分钟内没有任何数据传输（包括 Ping/Pong），则断开连接
		// 这有助于清理僵尸连接
		IdleTimeout: *idleTimeout,

		// 每 30 秒向客户端发送一次 Ping
		// 客户端收到 Ping 后会自动回复 Pong
		// 这有助于保持 NAT 映射活跃，并刷新 IdleTimeout
		PingInterval: *pingInterval,
	}

	parsedAddrs := parseAddrs(*addrs)
	log.Printf("启动服务器监听: %v", parsedAddrs)
	log.Printf("配置: IdleTimeout=%v, PingInterval=%v", opts.IdleTimeout, opts.PingInterval)

	// 3. 启动服务器
	if err := tcplayer.ListenAndServeTLS(parsedAddrs, *certFile, *keyFile, r, opts); err != nil {
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
