# flupoc-go

基于 TLS 的二进制路由库，提供类 HTTP 风格的 API 设计：

- **协议层**：`protocol/head` 和 `protocol/datagram` 定义 8 字节帧头（协议/类型/通道ID/数据长度）及数据报序列化；`protocol/service` 负责连接生命周期管理、数据报读写和请求分发。
- **路由层**：`router.Router` 支持 Method/Path 匹配与中间件，内置 Poculum 编解码。
- **传输层**：`transport` 仅负责 TLS TCP 连接的创建与接受，将连接交给协议层处理。
- **客户端**：`client` 封装 TLS 连接、数据报发送与响应解码，提供可复用的 `Client` 及类 HTTP 辅助方法。
- **命令行**：`cmd/main.go`（服务器）、`cmd/demo_client/main.go`（客户端示例）。

## 架构概览

```
┌─────────────────────────────────────────────────────────────┐
│                        应用层                                │
│                    router.Router                            │
│              (路由匹配、中间件、请求处理)                      │
├─────────────────────────────────────────────────────────────┤
│                       协议层                                 │
│               protocol/service                              │
│    (连接生命周期、数据报读写、心跳、空闲超时、请求分发)          │
├─────────────────────────────────────────────────────────────┤
│                       传输层                                 │
│                    transport                                │
│              (TLS 连接创建与接受)                             │
└─────────────────────────────────────────────────────────────┘
```

## 快速开始

### 服务器（命令行）
```powershell
go run ./cmd --addrs="127.0.0.1:5128" --cert="server.crt" --key="server.key"
```
- 必需参数：`--addrs`、`--cert`、`--key`；默认路由为 `POST /echo`。

### 客户端（命令行）
快速调试（跳过证书校验）：
```powershell
go run ./cmd/demo_client --addr=127.0.0.1:5128 --insecure=true --body="hello"
```
完整校验（含 CA/证书）：
```powershell
go run ./cmd/demo_client --addr=127.0.0.1:5128 ^
  --ca="ca.pem" --cert="client.crt" --key="client.key" ^
  --insecure=false --path=/echo --body="hello"
```

## 代码示例

### 最简服务器
```go
r := router.NewRouter()
r.Post("/echo", func(ctx *router.Context) (*router.Response, error) {
    return router.Bytes(ctx.RequestBody), nil
})

svc := service.New(r, service.Options{})
transport.ListenAndServeTLS([]string{"127.0.0.1:5128"}, certFile, keyFile, svc.Handle)
```

### 带配置的服务器
```go
svc := service.New(r, service.Options{
    IdleTimeout:  2 * time.Minute,  // 空闲超时断开
    PingInterval: 30 * time.Second, // 心跳保活
})
transport.ListenAndServeTLS(addrs, cert, key, svc.Handle)
```

### 最简客户端
```go
cli, _ := client.New(client.TLSConfig{Insecure: true})
resp, _ := cli.Post("127.0.0.1:5128", "/echo", []byte("hello"))
fmt.Println(string(resp.Body))
```

### 带配置的客户端
```go
cli, _ := client.New(client.TLSConfig{CAFile: "ca.pem"}, client.Options{
    DialTimeout:  5 * time.Second,
    ReadTimeout:  10 * time.Second,
    WriteTimeout: 10 * time.Second,
})
resp, _ := cli.Do("127.0.0.1:5128", "POST", "/echo", []byte("data"))
```

## API 参考

### 协议层配置 (`service.Options`)
| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `IdleTimeout` | `time.Duration` | `0` | 应用层无通信超时后断开（0=永不超时） |
| `PingInterval` | `time.Duration` | `0` | 服务器定期发送 PING；客户端自动回复 PONG，用于保活与空闲探测，0=不发送ping包 |

### 客户端配置 (`client.Options`)
| 字段 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `DialTimeout` | `time.Duration` | `10s` | 连接超时 |
| `ReadTimeout` | `time.Duration` | `30s` | 读响应超时 |
| `WriteTimeout` | `time.Duration` | `30s` | 写请求超时 |

- 客户端会自动回复服务器的 PING（发送 PONG），然后继续等待真正的响应。

### 客户端方法
```go
// 通用请求
func (c *Client) Do(addr, method, path string, body []byte) (*Response, error)

// 发出四种请求的方法
func (c *Client) Get(addr, path string) (*Response, error)
func (c *Client) Post(addr, path string, body []byte) (*Response, error)
func (c *Client) Put(addr, path string, body []byte) (*Response, error)
func (c *Client) Delete(addr, path string) (*Response, error)
```

### 路由响应辅助函数
```go
router.Bytes(data []byte) *Response           // 原始字节响应
router.Text(s string) *Response               // 纯文本响应
router.JSON(v any) (*Response, error)         // JSON 响应
router.Error(code int, msg string) *Response  // 错误响应
```

### 协议常量 (`protocol/head`)
```go
head.ProtocolID  = 0xCF  // Flupoc 协议标识符
head.MsgRequest  = 0x01  // 请求消息类型
head.MsgResponse = 0x02  // 响应消息类型
head.MsgPing     = 0x03  // PING 消息类型
head.MsgPong     = 0x04  // PONG 消息类型
```

## 目录结构
| 目录 | 说明 |
|------|------|
| `cmd/` | 服务器入口与客户端示例 |
| `transport/` | TLS TCP 传输层（仅负责连接创建与接受） |
| `protocol/service/` | 协议层连接服务（连接生命周期、数据报处理、心跳） |
| `protocol/head/` | 帧头定义 |
| `protocol/datagram/` | 数据报序列化 |
| `client/` | TLS 客户端（Client / Do / Get / Post） |
| `router/` | 路由与 Poculum 编解码 |
| `poculum/` | Poculum 序列化格式 |

## TLS 与证书
- **服务器**：必须提供证书和私钥（强制 TLS）。
- **客户端**：受信任 CA 可直接校验；自签/私有 CA 需提供 `--ca` 或导入系统信任。调试时可用 `--insecure=true` 跳过校验（仍加密传输）。
