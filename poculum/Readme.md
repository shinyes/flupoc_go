# Poculum - Go 实现

## 概述

Poculum 是一种数据交换格式，支持：
- `int`、`uint`、`float32`、`float64`
- `list`（元素类型可不同）
- `map`（字符串键，任意支持的值类型）

## 特性

- **高性能**：利用 Go 编译器优化与内存管理
- **零依赖**：仅使用 Go 标准库
- **反射支持**：自动处理接口类型
- **布尔值支持**：正确序列化 true/false，跨语言兼容
- **接口友好**：支持 `interface{}`，但具体类型限于支持的数据类型

## 支持的数据类型

### 基本类型
| 类型 | 说明 |
|------|------|
| `int`、`int8/16/32/64`、`uint`、`uint8/16/32/64` | 整数 - 自动选择最优编码 |
| `float32`、`float64` | 高精度浮点数 |
| `bool` | 布尔值（true/false） |
| `string` | UTF-8 编码字符串 |
| `[]byte` | 原始二进制数据 |
| `nil` | 空值 |

### 复合类型
| 类型 | 说明 |
|------|------|
| `[]T` | 任意类型切片 |
| `[N]T` | 固定长度数组 |
| `map[string]T` | 字符串键映射 |
| `interface{}` | 任意类型（限于支持的类型） |

## 快速开始

```go
package main

import (
    "fmt"
    "log"

    "github.com/cykyes/flupoc-go/poculum"
)

func main() {
    // 构建数据结构
    list := []any{1, "2", nil}
    data := map[string]any{
        "integer":       int32(42),
        "float":         float64(3.14159),
        "boolean_true":  true,
        "boolean_false": false,
        "string":        "Hello, 世界!",
        "unicode":       "🌟✨🚀💫",
        "bytes":         []byte("binary data"),
        "null":          nil,
        "list":          list,
    }

    // 序列化
    encoded, err := poculum.DumpPoculum(data)
    if err != nil {
        log.Fatal("序列化失败:", err)
    }
    fmt.Printf("序列化大小: %d 字节\n", len(encoded))

    // 反序列化
    decoded, err := poculum.LoadPoculum(encoded)
    if err != nil {
        log.Fatal("反序列化失败:", err)
    }
    fmt.Printf("反序列化结果: %+v\n", decoded)
}
```

## API 参考

### 序列化
```go
// 将任意支持的值序列化为 Poculum 格式
func DumpPoculum(v any) ([]byte, error)

// 使用自定义限制
func WithLimits(maxDepth, maxSize int) *Poculum
func (p *Poculum) Dump(v any) ([]byte, error)
```

### 反序列化
```go
// 将 Poculum 字节反序列化为 Go 值
func LoadPoculum(data []byte) (any, error)

// 使用自定义限制
func (p *Poculum) Load(data []byte) (any, error)
```

## 与 Router 集成

`router` 包自动使用 Poculum 进行请求/响应编解码：

```go
r := router.NewRouter()

// 请求体已从 Poculum 解码
r.Post("/data", func(ctx *router.Context) (*router.Response, error) {
    // ctx.RequestBody 包含原始字节
    // ctx.RequestData 包含解码后的 Poculum 数据
    return router.JSON(ctx.RequestData), nil
})
```
