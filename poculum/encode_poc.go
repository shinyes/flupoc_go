package poculum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"reflect"
	"unicode/utf8"
)

// 编码值到缓冲区
func (poc *Poculum) encodeValue(value any, buf *bytes.Buffer, depth int) error {
	if depth > poc.maxRecursionDepth {
		return newError("MaxRecursionDepth", "Maximum recursion depth exceeded")
	}

	switch v := value.(type) {
	case uint8:
		buf.WriteByte(typeUInt8)
		buf.WriteByte(v)
	case uint16:
		buf.WriteByte(typeUInt16)
		binary.Write(buf, binary.BigEndian, v)
	case uint32:
		buf.WriteByte(typeUInt32)
		binary.Write(buf, binary.BigEndian, v)
	case uint64:
		buf.WriteByte(typeUInt64)
		binary.Write(buf, binary.BigEndian, v)
	case int8:
		buf.WriteByte(typeInt8)
		buf.WriteByte(byte(v))
	case int16:
		buf.WriteByte(typeInt16)
		binary.Write(buf, binary.BigEndian, v)
	case int32:
		buf.WriteByte(typeInt32)
		binary.Write(buf, binary.BigEndian, v)
	case int64:
		buf.WriteByte(typeInt64)
		binary.Write(buf, binary.BigEndian, v)
	case int:
		// Go 的 int 类型，转换为适当的整数类型
		if v >= 0 {
			if v <= math.MaxUint32 {
				return poc.encodeValue(uint32(v), buf, depth)
			} else {
				return poc.encodeValue(uint64(v), buf, depth)
			}
		} else {
			if v >= math.MinInt32 {
				return poc.encodeValue(int32(v), buf, depth)
			} else {
				return poc.encodeValue(int64(v), buf, depth)
			}
		}
	case uint:
		// Go 的 uint 类型
		if v <= math.MaxUint32 {
			return poc.encodeValue(uint32(v), buf, depth)
		} else {
			return poc.encodeValue(uint64(v), buf, depth)
		}
	case float32:
		buf.WriteByte(typeFloat32)
		binary.Write(buf, binary.BigEndian, v)
	case float64:
		buf.WriteByte(typeFloat64)
		binary.Write(buf, binary.BigEndian, v)
	case string:
		return poc.encodeString(v, buf)
	case []any: // 这里对应的是序列化数组的部分
		return poc.encodeArray(v, buf, depth)
	case map[string]any:
		return poc.encodeMap(v, buf, depth)
	case []byte:
		return poc.encodeBytes(v, buf)
	case bool:
		// 布尔值
		if v {
			buf.WriteByte(typeTrue)
		} else {
			buf.WriteByte(typeFalse)
		}
	case nil:
		return buf.WriteByte(typeNil)
	default:
		// 使用反射处理其他类型
		return poc.encodeWithReflection(value, buf, depth)
	}

	return nil
}

// encodeWithReflection 使用反射编码未知类型
func (poc *Poculum) encodeWithReflection(value any, buf *bytes.Buffer, depth int) error {
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Bool:
		// 处理布尔类型，保持与主分支一致
		if rv.Bool() {
			buf.WriteByte(typeTrue)
		} else {
			buf.WriteByte(typeFalse)
		}
		return nil
	case reflect.Slice:
		// 处理切片类型
		length := rv.Len()
		values := make([]any, length)
		for i := 0; i < length; i++ {
			values[i] = rv.Index(i).Interface()
		}
		return poc.encodeArray(values, buf, depth)
	case reflect.Map:
		// 处理映射类型
		if rv.Type().Key().Kind() != reflect.String {
			return newError("UnsupportedType", "Map keys must be strings")
		}
		values := make(map[string]any)
		for _, key := range rv.MapKeys() {
			keyStr := key.String()
			value := rv.MapIndex(key).Interface()
			values[keyStr] = value
		}
		return poc.encodeMap(values, buf, depth)
	default:
		return newError("UnsupportedType", fmt.Sprintf("Unsupported type: %T", value))
	}
}

// encodeString 编码字符串
func (poc *Poculum) encodeString(s string, buf *bytes.Buffer) error {
	data := []byte(s)
	length := len(data)

	if length > poc.maxStringSize {
		return newError("DataTooLarge", fmt.Sprintf("String too long: %d bytes (max %d)", length, poc.maxStringSize))
	}

	if !utf8.Valid(data) {
		return newError("Utf8Error", "Invalid UTF-8 string")
	}

	if length <= 15 {
		// fixstring
		buf.WriteByte(typeFixStringBase + byte(length))
		buf.Write(data)
	} else if length <= 0xFFFF {
		// string16
		buf.WriteByte(typeString16)
		binary.Write(buf, binary.BigEndian, uint16(length))
		buf.Write(data)
	} else {
		// string32
		buf.WriteByte(typeString32)
		binary.Write(buf, binary.BigEndian, uint32(length))
		buf.Write(data)
	}

	return nil
}

// encodeArray 编码数组
func (poc *Poculum) encodeArray(arr []any, buf *bytes.Buffer, depth int) error {
	length := len(arr)

	if length > poc.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Array too long: %d items (max %d)", length, poc.maxContainerItems))
	}

	// 先把类型字节与长度写入到字节缓冲区
	if length <= 15 {
		// fixlist
		buf.WriteByte(typeFixListBase + byte(length))
	} else if length <= 0xFFFF {
		// list16
		buf.WriteByte(typeList16)
		binary.Write(buf, binary.BigEndian, uint16(length))
	} else {
		// list32
		buf.WriteByte(typeList32)
		binary.Write(buf, binary.BigEndian, uint32(length))
	}

	// 再逐个序列化数组中的项
	for _, item := range arr {
		err := poc.encodeValue(item, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeMap 编码对象
func (poc *Poculum) encodeMap(obj map[string]any, buf *bytes.Buffer, depth int) error {
	length := len(obj)

	if length > poc.maxContainerItems {
		return newError("DataTooLarge", fmt.Sprintf("Object too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	// 先把类型字节写入到字节缓冲区
	if length <= 15 {
		// fixmap
		buf.WriteByte(typeFixMapBase + byte(length))
	} else if length <= 0xFFFF {
		// map16
		buf.WriteByte(typeMap16)
		binary.Write(buf, binary.BigEndian, uint16(length))
	} else {
		// map32
		buf.WriteByte(typeMap32)
		binary.Write(buf, binary.BigEndian, uint32(length))
	}
	// 再逐个序列化键与值
	for key, value := range obj {
		err := poc.encodeString(key, buf)
		if err != nil {
			return err
		}
		err = poc.encodeValue(value, buf, depth+1)
		if err != nil {
			return err
		}
	}

	return nil
}

// encodeBytes 编码字节数据
func (poc *Poculum) encodeBytes(data []byte, buf *bytes.Buffer) error {
	length := len(data)

	if length <= 0xFF {
		// bytes8
		buf.WriteByte(typeBytes8)
		buf.WriteByte(byte(length))
		buf.Write(data)
	} else if length <= 0xFFFF {
		// bytes16
		buf.WriteByte(typeBytes16)
		binary.Write(buf, binary.BigEndian, uint16(length))
		buf.Write(data)
	} else {
		// bytes32
		buf.WriteByte(typeBytes32)
		binary.Write(buf, binary.BigEndian, uint32(length))
		buf.Write(data)
	}

	return nil
}

// 序列化值为字节数组
func (poc *Poculum) dump(value any) ([]byte, error) {
	var buf bytes.Buffer
	err := poc.encodeValue(value, &buf, 0)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func LoadPoculum(data []byte) (any, error) {
	mb := NewPoculum()
	return mb.load(data)
}
