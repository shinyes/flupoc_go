# Echo 测试程序使用说明

## 概述

这个示例包含一个回显服务端和一个测试客户端，用于测试 Flupoc 协议的数据传输功能。

## 回显服务端 (echo_server)

位置：`cmd/echo_server/main.go`

### 功能
- 接收客户端发送的二进制数据
- 原样返回接收到的数据
- 支持通过 `/echo` 路径访问

### 使用方法

```bash
go run .\cmd\echo_server\main.go --addrs="127.0.0.1:5128" --cert="path/to/cert.crt" --key="path/to/key.key"
```

### 参数说明
- `--addrs`: 监听地址（默认：127.0.0.1:5128），支持逗号分隔的多个地址
- `--cert`: TLS 证书文件路径（必填）
- `--key`: TLS 私钥文件路径（必填）

## 测试客户端 (echo_client)

位置：`cmd/echo_client/main.go`

### 功能
- 每隔 0~60 秒（随机）发送一次请求
- 每次发送 0~20MB 的随机数据
- 验证服务端返回的数据是否与发送的数据相同
- 将测试结果记录到 CSV 文件中（追加模式）

### CSV 记录内容
- 发送日期时间（东八区/UTC+8）
- 数据是否相同（是/否）
- 往返时间（毫秒）
- 数据大小（字节）
- 错误信息（如有）

### 使用方法

```bash
# 使用 CA 证书验证服务器
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --ca="path/to/ca.crt"

# 跳过证书验证（仅用于测试）
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --insecure

# 使用客户端证书（mTLS）
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --ca="path/to/ca.crt" --cert="path/to/client.crt" --key="path/to/client.key"

# 限制运行次数
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --ca="path/to/ca.crt" --runs=10
```

### 参数说明
- `--addr`: 服务器地址（默认：127.0.0.1:5128）
- `--ca`: CA 证书文件路径，用于验证服务器证书
- `--cert`: 客户端证书文件路径（可选，用于 mTLS）
- `--key`: 客户端私钥文件路径（可选，用于 mTLS）
- `--insecure`: 跳过服务器证书验证（不推荐在生产环境使用）
- `--runs`: 运行次数，0 表示无限循环（默认：0）

### CSV 输出文件

客户端会在当前目录生成 `echo_test_log.csv` 文件，文件包含 UTF-8 BOM 以便 Excel 正确识别中文。

多次运行程序时，新的测试数据会追加到同一个 CSV 文件中，不会覆盖之前的记录。

## 完整使用示例

### 1. 启动服务端

```bash
go run .\cmd\echo_server\main.go --addrs="127.0.0.1:5128" --cert="d:\gencert-data\certs\server.crt" --key="d:\gencert-data\certs\server.key"
```

### 2. 运行客户端（另一个终端）

```bash
# 运行 10 次测试
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --ca="d:\gencert-data\certs\ca.crt" --runs=10

# 或无限循环运行（按 Ctrl+C 停止）
go run .\cmd\echo_client\main.go --addr="127.0.0.1:5128" --ca="d:\gencert-data\certs\ca.crt"
```

### 3. 查看测试结果

打开 `echo_test_log.csv` 文件查看测试结果，可以使用 Excel 或任何文本编辑器打开。

## 注意事项

- 确保服务端和客户端使用的证书文件路径正确
- 服务端必须先启动，客户端才能连接
- CSV 文件会自动创建，如果文件已存在则追加数据
- 客户端发送的随机数据大小范围是 0~20MB，包括边界值
- 等待时间范围是 0~60 秒，包括边界值
- 所有时间记录都使用东八区时区（UTC+8）
