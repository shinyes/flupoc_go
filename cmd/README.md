# 命令行入口说明

## cmd/main.go（服务器）

TLS TCP 服务器，支持路由与多地址监听。

### 用法
```powershell
go run ./cmd --addrs="127.0.0.1:5128" --cert="server.crt" --key="server.key"
```

### 参数说明
| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `--addrs` | 是 | `127.0.0.1:5128` | 逗号分隔的监听地址（支持 IPv4/IPv6） |
| `--cert` | 是 | - | 服务器证书路径 |
| `--key` | 是 | - | 服务器私钥路径 |
| `--tls` | - | `true` | 始终为 true（协议要求 TLS） |

### 内置路由
- `POST /echo` - 回显请求体
- 可在启动前修改或扩展路由注册逻辑

### 示例
```powershell
go run ./cmd --addrs="127.0.0.1:5128,[::]:8443" --cert="D:/certs/server.crt" --key="D:/certs/server.key"
```

---

## cmd/demo_client/main.go（客户端）

演示客户端，向服务器发送单次请求。

### 用法
```powershell
go run ./cmd/demo_client --addr=127.0.0.1:5128 --insecure=true --body="hello"
```

### 参数说明
| 参数 | 必需 | 默认值 | 说明 |
|------|------|--------|------|
| `--addr` | 是 | `127.0.0.1:5128` | 服务器地址（host:port） |
| `--method` | 否 | `POST` | HTTP 风格方法 |
| `--path` | 否 | `/echo` | 请求路径 |
| `--body` | 否 | `hello demo` | 请求体字符串 |
| `--ca` | 否 | - | CA 证书（用于校验服务器） |
| `--cert` | 否 | - | 客户端证书（双向 TLS） |
| `--key` | 否 | - | 客户端私钥（双向 TLS） |
| `--insecure` | 否 | `false` | 跳过服务器证书校验 |

### 示例

**调试模式（跳过校验）：**
```powershell
go run ./cmd/demo_client --addr=127.0.0.1:5128 --insecure=true --body="hello"
```

**完整 TLS 校验：**
```powershell
go run ./cmd/demo_client --addr=127.0.0.1:5128 ^
  --ca="D:/certs/ca.pem" ^
  --cert="D:/certs/client.crt" ^
  --key="D:/certs/client.key" ^
  --insecure=false --path=/echo --body="hello demo"
```

---

## 客户端超时配置

直接使用客户端库时，可通过 `client.Options` 配置超时：

```go
cli, _ := client.New(client.TLSConfig{CAFile: "ca.pem"}, client.Options{
    DialTimeout:  5 * time.Second,   // 连接超时（默认：10s）
    ReadTimeout:  10 * time.Second,  // 读响应超时（默认：30s）
    WriteTimeout: 10 * time.Second,  // 写请求超时（默认：30s）
})
```
