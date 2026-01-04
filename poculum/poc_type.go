package poculum

import (
	"fmt"
	"math"
)

// 以下定义类型标识符常量，长度都是一个字节
/*
	关于名称中带有 Fix 的类型

如果名称中带有 fix 则代表其中的元素个数<=15，专门设立这个类型是为了当元素个数比较小时，还要用去存储长度字段，
这种类型的长度字段是存储在字节的低位的，所以名称中才会带有Base，
例如如果某个值的类型字节为 0x3F，则代表这是一个字符串，则这个字符串占用的字节数为 15，
对于 fix 的 List 和 Map，类型字节的低位代表的是其中的元素个数
*/
const (
	typeUInt8  = 0x01
	typeUInt16 = 0x02
	typeUInt32 = 0x03
	typeUInt64 = 0x04
	// typeUInt128 = 0x05 // 暂时不使用

	typeInt8  = 0x11
	typeInt16 = 0x12
	typeInt32 = 0x13
	typeInt64 = 0x14
	// typeInt128 = 0x15 // 暂时不使用

	typeFloat32 = 0x21
	typeFloat64 = 0x22

	typeFixStringBase = 0x30
	typeString16      = 0x41
	typeString32      = 0x42

	typeFixListBase = 0x50
	typeList16      = 0x61
	typeList32      = 0x62

	typeFixMapBase = 0x70
	typeMap16      = 0x81
	typeMap32      = 0x82

	typeBytes8  = 0x91
	typeBytes16 = 0x92
	typeBytes32 = 0x93

	typeTrue  = 0xA0
	typeFalse = 0xA1
	// typeUnkown = 0xA2 // 暂不使用
	typeNil = 0xA3
)

// 安全限制常量
const (
	maxRecursionDepth = math.MaxUint32 // list、map的最大嵌套深度，4G层
	maxStringSize     = math.MaxUint32 // 默认情况下字符串最大字节数 4GB
	maxContainerItems = math.MaxUint32 // 默认情况下 list、map中的最多元素数量，4G个
)

// Poculum 编码器/解码器
type Poculum struct {
	maxRecursionDepth int
	maxStringSize     int
	maxContainerItems int
}

// PoculumError 错误类型
type PoculumError struct {
	Type    string
	Message string
}

func (e *PoculumError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// 错误构造函数
func newError(errType, message string) *PoculumError {
	return &PoculumError{Type: errType, Message: message}
}

// NewPoculum 创建新的 Poculum 实例
func NewPoculum() *Poculum {
	return &Poculum{
		maxRecursionDepth: maxRecursionDepth,
		maxStringSize:     maxStringSize,
		maxContainerItems: maxContainerItems,
	}
}

// WithLimits 创建具有自定义限制的 Poculum 实例
func WithLimits(maxRecursion, maxStringSize, maxContainerItems int) *Poculum {
	return &Poculum{
		maxRecursionDepth: maxRecursion,
		maxStringSize:     maxStringSize,
		maxContainerItems: maxContainerItems,
	}
}
