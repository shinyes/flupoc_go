# Flupoc-Go 项目文档

**Flupoc-Go** 是一个基于 TLS 的高性能二进制路由库。它结合了自定义二进制协议（Flupoc）与高效序列化格式（Poculum），提供了类似 HTTP 框架（如 Gin/Echo）的开发体验，但运行在更轻量、更紧凑的 TCP 长连接之上。

## 1. 项目架构概览

项目采用分层架构设计，各层职责清晰：

| 层级 | 目录 | 说明 |
| :--- | :--- | :--- |
| **应用层 (Router)** | `router/` | 提供路由分发、中间件、上下文管理（类似 HTTP 框架）。 |
| **传输层 (TCP Layer)** | `tcp_layer/` | 基于 TLS 的 TCP 服务器，负责连接管理、心跳保活、数据报收发。 |
| **协议层 (Protocol)** | `protocol/` | 定义二进制帧格式（Head）与数据报（Datagram）结构。 |
| **序列化层 (Poculum)** | `poculum/` | 自研二进制序列化格式，支持基础类型与嵌套结构。 |
| **客户端 (Client)** | `client/` | 封装了连接池、TLS 握手、协议编解码的客户端库。 |

---

## 2. 快速开始

### 2.1 环境要求
- Go 1.25+
- TLS 证书与私钥（用于服务器和客户端）

### 2.2 编写第一个服务器

创建一个简单的回显服务器：

```go
package main

import (
    "log"
    "github.com/cykyes/flupoc-go/router"
    tcplayer "github.com/cykyes/flupoc-go/tcp_layer"
)

func main() {
    // 1. 创建路由器
    r := router.NewRouter()

    // 2. 注册路由
    r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
        log.Printf("收到请求: %s", string(ctx.RequestBody))
        return router.Bytes(ctx.RequestBody), nil
    })

    // 3. 启动 TLS 服务器
    // 需准备 server.crt 和 server.key
    err := tcplayer.ListenAndServeTLS(
        []string{"127.0.0.1:5128"}, 
        "server.crt", 
        "server.key", 
        r, 
        nil, // 使用默认配置
    )
    if err != nil {
        log.Fatal(err)
    }
}
```

### 2.3 编写客户端

```go
package main

import (
    "fmt"
    "log"
    "github.com/cykyes/flupoc-go/client"
)

func main() {
    // 1. 创建客户端 (跳过证书校验用于测试)
    cli, _ := client.New(client.Options{
        Insecure: true,
    })

    // 2. 发送请求
    resp, err := cli.Post("127.0.0.1:5128", "/echo", []byte("Hello Flupoc"))
    if err != nil {
        log.Fatal(err)
    }

    // 3. 处理响应
    fmt.Printf("响应状态: %d\n", resp.StatusCode)
    fmt.Printf("响应内容: %s\n", resp.Body)
}
```

---

## 3. 核心组件详解

### 3.1 协议层 (Protocol)

Flupoc 协议基于 TCP，所有数据被封装为 **帧 (Frame)**。

#### 帧头结构 (8 字节)
| 偏移 | 长度 | 字段 | 说明 |
| :--- | :--- | :--- | :--- |
| 0 | 1 | `Protocol` | 固定为 `0xCF` |
| 1 | 1 | `Type` | 消息类型 |
| 2 | 2 | `ChannelID` | 通道 ID (用于多路复用，目前主要用 1) |
| 4 | 4 | `DataLength` | 数据载荷长度 (大端序) |

#### 消息类型
- `MsgPing (0x01)` / `MsgPong (0x02)`: 心跳保活，由传输层自动处理。
- `MsgRequest (0x03)`: 客户端请求，载荷为 Poculum 序列化的 Map。
- `MsgResponse (0x04)`: 服务器响应，载荷为 Poculum 序列化的 Map。

### 3.2 序列化层 (Poculum)

Poculum 是项目内置的二进制序列化格式，位于 `poculum/` 目录。

- **特点**: 紧凑、强类型、支持嵌套。
- **支持类型**:
    - 基础: `int`, `uint`, `float`, `bool`, `string`, `[]byte`, `nil`
    - 复合: `[]any` (列表), `map[string]any` (字典)

**使用示例**:
```go
import "github.com/cykyes/flupoc-go/poculum"

data := map[string]any{
    "id": 1001,
    "name": "flupoc",
    "tags": []any{"fast", "secure"},
}

// 序列化
bytes, _ := poculum.DumpPoculum(data)

// 反序列化
obj, _ := poculum.LoadPoculum(bytes)
```

### 3.3 路由层 (Router)

`router` 包提供了类似 Web 框架的 API。

#### 路由注册
支持标准 HTTP 方法：
```go
r.Get("/users/{id}", getUserHandler)
r.Post("/users", createUserHandler)
r.Put("/users/{id}", updateUserHandler)
r.Delete("/users/{id}", deleteUserHandler)
```

#### 路径参数
使用 `{param}` 语法定义参数：
```go
r.Get("/files/{name}", func(ctx *router.Context) (*router.Response, error) {
    fileName := ctx.Param("name")
    return router.Text("File: " + fileName), nil
})
```

#### 中间件 (Middleware)
支持全局中间件和路由组中间件：
```go
// 全局日志中间件
r.Use(func(next router.HandlerFunc) router.HandlerFunc {
    return func(ctx *router.Context) (*router.Response, error) {
        log.Println("Request:", ctx.Path)
        return next(ctx)
    }
})

// 路由组
api := r.Group("/api")
api.Use(authMiddleware) // 仅对 /api 下生效
api.Get("/info", infoHandler)
```

### 3.4 传输层 (TCP Layer)

`tcp_layer` 负责底层的网络交互。

**服务器配置 (`ServerOptions`)**:
- `IdleTimeout`: 连接空闲超时时间，超时自动断开。
- `PingInterval`: 服务器主动发送 Ping 的间隔，客户端会自动回复 Pong。

```go
opts := &tcplayer.ServerOptions{
    IdleTimeout:  5 * time.Minute,
    PingInterval: 30 * time.Second,
}
tcplayer.ListenAndServeTLS(addrs, cert, key, router, opts)
```

---

## 4. 命令行工具

项目提供了开箱即用的命令行工具，位于 `cmd/` 目录下。

### 4.1 服务器 (`cmd/main.go`)
启动一个通用服务器，默认包含 `/echo` 路由。

```powershell
# 启动服务器
go run ./cmd --addrs="0.0.0.0:5128" --cert="cert.pem" --key="key.pem"
```

### 4.2 演示客户端 (`cmd/demo_client/main.go`)
用于调试和测试服务器接口。

```powershell
# 发送请求 (忽略证书校验)
go run ./cmd/demo_client --addr="127.0.0.1:5128" --insecure=true --body="hello"

# 发送请求 (带完整证书校验)
go run ./cmd/demo_client \
  --addr="127.0.0.1:5128" \
  --ca="ca.pem" \
  --cert="client.crt" \
  --key="client.key" \
  --path="/echo" \
  --body="test data"
```

---

## 5. API 参考

### Router Context
| 方法 | 说明 |
| :--- | :--- |
| `Param(key)` | 获取路径参数 (如 `/user/{id}`) |
| `Query(key)` | 获取查询参数 (如 `?page=1`) |
| `RequestBody` | 原始请求体字节 |

### Router Response 辅助函数
| 函数 | 说明 |
| :--- | :--- |
| `router.OK(body)` | 返回 200 OK (通用) |
| `router.Text(str)` | 返回文本响应 |
| `router.JSON(obj)` | 返回 JSON 响应 |
| `router.Bytes(data)` | 返回二进制响应 |
| `router.Error(code, msg)` | 返回错误响应 |

### Client Options
| 字段 | 说明 |
| :--- | :--- |
| `DialTimeout` | 连接超时 (默认 5s) |
| `ReadTimeout` | 读取响应超时 (默认 5s) |
| `WriteTimeout` | 发送请求超时 (默认 5s) |
| `Insecure` | 是否跳过 TLS 证书校验 |
| `CAFile` | 自定义 CA 证书路径 |

---

## 6. 常见问题

**Q: 为什么需要 Poculum？**
A: Poculum 专为 Go 语言优化，比 JSON 更紧凑，比 Protobuf 更灵活（无需预定义 Schema），适合动态数据交换。

**Q: 客户端如何处理心跳？**
A: `client.Client` 内部实现了自动处理机制。当收到服务器的 `MsgPing` 时，客户端会立即回复 `MsgPong` 并重置读取超时，这对上层调用者是透明的。

**Q: 支持非 TLS 模式吗？**
A: 不支持。为了安全性，Flupoc 协议强制要求 TLS 加密。测试时可使用自签名证书并开启 `Insecure: true`。
