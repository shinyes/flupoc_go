package poculum

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode/utf8"
)

// 从字节数组反序列化值
func (poc *Poculum) load(data []byte) (any, error) {
	if len(data) == 0 {
		return nil, nil
	}

	reader := bytes.NewReader(data)
	return poc.decodeValue(reader, 0)
}

// decodeValue 从bytes.Reader中解码出值
func (poc *Poculum) decodeValue(reader *bytes.Reader, depth int) (any, error) {
	if depth > poc.maxRecursionDepth {
		return nil, newError("MaxRecursionDepth", "Maximum recursion depth exceeded while parsing nested structure")
	}

	typeByte, err := reader.ReadByte()
	if err != nil {
		return nil, newError("InsufficientData", "No type byte")
	}

	switch typeByte {
	case typeUInt8:
		var value uint8
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint8")
		}
		return value, nil
	case typeUInt16:
		var value uint16
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint16")
		}
		return value, nil
	case typeUInt32:
		var value uint32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint32")
		}
		return value, nil
	case typeUInt64:
		var value uint64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "uint64")
		}
		return value, nil
	case typeInt8:
		var value int8
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int8")
		}
		return value, nil
	case typeInt16:
		var value int16
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int16")
		}
		return value, nil
	case typeInt32:
		var value int32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int32")
		}
		return value, nil
	case typeInt64:
		var value int64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "int64")
		}
		return value, nil
	case typeFloat32:
		var value float32
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "float32")
		}
		return value, nil
	case typeFloat64:
		var value float64
		err := binary.Read(reader, binary.BigEndian, &value)
		if err != nil {
			return nil, newError("InsufficientData", "float64")
		}
		return value, nil
	case typeTrue:
		return true, nil
	case typeFalse:
		return false, nil
	case typeNil:
		return nil, nil
	default:
		// 处理字符串类型
		if typeByte >= typeFixStringBase && typeByte <= typeFixStringBase+15 {
			length := int(typeByte - typeFixStringBase)
			return poc.decodeString(reader, length)
		}
		if typeByte == typeString16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string16 length")
			}
			return poc.decodeString(reader, int(length))
		}
		if typeByte == typeString32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "string32 length")
			}
			if int(length) > poc.maxStringSize {
				return nil, newError("DataTooLarge", fmt.Sprintf("String32 length too large: %d", length))
			}
			return poc.decodeString(reader, int(length))
		}

		// 处理数组类型
		if typeByte >= typeFixListBase && typeByte <= typeFixListBase+15 {
			length := int(typeByte - typeFixListBase)
			return poc.decodeArray(reader, length, depth)
		}
		if typeByte == typeList16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list16 length")
			}
			return poc.decodeArray(reader, int(length), depth)
		}
		if typeByte == typeList32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "list32 length")
			}
			return poc.decodeArray(reader, int(length), depth)
		}

		// 处理对象类型
		if typeByte >= typeFixMapBase && typeByte <= typeFixMapBase+15 {
			length := int(typeByte - typeFixMapBase)
			return poc.decodeMap(reader, length, depth)
		}
		if typeByte == typeMap16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map16 length")
			}
			return poc.decodeMap(reader, int(length), depth)
		}
		if typeByte == typeMap32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "map32 length")
			}
			return poc.decodeMap(reader, int(length), depth)
		}

		// 处理字节数据类型
		if typeByte == typeBytes8 {
			var length uint8
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes8 length")
			}
			return poc.decodeBytes(reader, int(length))
		}
		if typeByte == typeBytes16 {
			var length uint16
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes16 length")
			}
			return poc.decodeBytes(reader, int(length))
		}
		if typeByte == typeBytes32 {
			var length uint32
			err := binary.Read(reader, binary.BigEndian, &length)
			if err != nil {
				return nil, newError("InsufficientData", "bytes32 length")
			}
			return poc.decodeBytes(reader, int(length))
		}

		return nil, newError("UnknownTypeId", fmt.Sprintf("Unknown type identifier: 0x%02x", typeByte))
	}
}

// decodeString 解码字符串
func (poc *Poculum) decodeString(reader *bytes.Reader, length int) (string, error) {
	if length == 0 {
		return "", nil
	}

	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil || n != length {
		return "", newError("InsufficientData", "string data")
	}

	if !utf8.Valid(data) {
		return "", newError("Utf8Error", "Invalid UTF-8 string")
	}

	return string(data), nil
}

// decodeArray 解码数组
func (poc *Poculum) decodeArray(reader *bytes.Reader, length int, depth int) ([]any, error) {
	if length > poc.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Array length too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	arr := make([]any, length)
	for i := 0; i < length; i++ {
		value, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		arr[i] = value
	}

	return arr, nil
}

// decodeMap 解码对象
func (poc *Poculum) decodeMap(reader *bytes.Reader, length int, depth int) (map[string]any, error) {
	if length > poc.maxContainerItems {
		return nil, newError("DataTooLarge", fmt.Sprintf("Object length too large: %d items (max %d)", length, poc.maxContainerItems))
	}

	obj := make(map[string]any)
	for i := 0; i < length; i++ {
		// 解码键
		keyValue, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		key, ok := keyValue.(string)
		if !ok {
			return nil, newError("UnsupportedType", "Object key must be string")
		}

		// 解码值
		value, err := poc.decodeValue(reader, depth+1)
		if err != nil {
			return nil, err
		}
		obj[key] = value
	}

	return obj, nil
}

// decodeBytes 解码字节数据
func (poc *Poculum) decodeBytes(reader *bytes.Reader, length int) ([]byte, error) {
	data := make([]byte, length)
	n, err := reader.Read(data)
	if err != nil || n != length {
		return nil, newError("InsufficientData", "bytes data")
	}

	return data, nil
}

func DumpPoculum(value any) ([]byte, error) {
	poc := NewPoculum()
	return poc.dump(value)
}
