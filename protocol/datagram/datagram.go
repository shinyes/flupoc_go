package datagram

import (
	"bytes"
	"encoding/binary"
	"errors"
	"strconv"

	"github.com/cespare/xxhash/v2"
	protocol "github.com/cykyes/flupoc-go/protocol/head"
)

type Datagram struct {
	Head *protocol.Head
	Data []byte
}

// 在 NewDatagram 函数中被调用用于生成校验码
func (d *Datagram) generateCheckCode() (*Datagram, error) {
	headSerialized, err := d.Head.SerializeWithoutCheckCode()
	if err != nil {
		return d, err
	}

	// 计算校验码（对头部信息和数据部分进行哈希）
	hasher := xxhash.New()
	hasher.Write(headSerialized)
	if len(d.Data) > 0 {
		hasher.Write(d.Data)
	}

	// 截取64位哈希值的前32位作为校验码
	d.Head.Check_code = uint32(hasher.Sum64())
	return d, nil
}

// 创建数据报结构体
func NewDatagram(channel_id uint16, msg_type uint8, data []byte) (*Datagram, error) {
	head := &protocol.Head{
		Protocol_id: protocol.FP_PROTOCOL_ID,
		Msg_type:    msg_type,
		Channel_id:  channel_id,
		Data_length: uint32(len(data)),
	}

	d := &Datagram{Head: head, Data: data}
	d, err := d.generateCheckCode()
	return d, err
}

// 序列化数据报
func (d *Datagram) Serialize() ([]byte, error) {
	buf := bytes.NewBuffer(make([]byte, 0, int(protocol.DATAGRAM_MIN_LEN)+len(d.Data)))
	// 序列化Head部分（共12字节）
	if err := binary.Write(buf, binary.BigEndian, d.Head.Protocol_id); err != nil {
		return nil, errors.New("序列化Protocol_id失败: " + err.Error())
	}
	if err := binary.Write(buf, binary.BigEndian, d.Head.Msg_type); err != nil {
		return nil, errors.New("序列化Msg_type失败: " + err.Error())
	}
	if err := binary.Write(buf, binary.BigEndian, d.Head.Channel_id); err != nil {
		return nil, errors.New("序列化Channel_id失败: " + err.Error())
	}
	if err := binary.Write(buf, binary.BigEndian, d.Head.Check_code); err != nil {
		return nil, errors.New("序列化Check_code失败: " + err.Error())
	}
	if err := binary.Write(buf, binary.BigEndian, d.Head.Data_length); err != nil {
		return nil, errors.New("序列化Length失败: " + err.Error())
	}
	// 写入数据部分
	if len(d.Data) > 0 {
		if _, err := buf.Write(d.Data); err != nil {
			return nil, errors.New("序列化数据部分失败: " + err.Error())
		}
	}
	return buf.Bytes(), nil
}

// 反序列化数据报
func Deserialize(datagramBytes []byte) (*Datagram, error) {
	// 1. 校验最小长度（至少包含Head，现在是12字节：8字节基础头部+4字节校验码）
	if len(datagramBytes) < int(protocol.DATAGRAM_MIN_LEN) {
		return nil, errors.New("字节流长度不足，Flupoc数据报至少12字节，实际为" + strconv.Itoa(len(datagramBytes)))
	}

	// 2. 反序列化Head
	head, err := protocol.Deserialize(datagramBytes[:protocol.DATAGRAM_MIN_LEN])
	if err != nil {
		return nil, err
	}

	// 3. 校验数据部分长度是否匹配
	if int(len(datagramBytes)-protocol.DATAGRAM_MIN_LEN) != int(head.Data_length) {
		return nil, errors.New("数据部分的长度不等于Head.length，Head.Length=" + strconv.FormatUint(uint64(head.Data_length), 10) +
			"，数据部分长度=" + strconv.Itoa(len(datagramBytes)-protocol.DATAGRAM_MIN_LEN) +
			"，数据报字节流总长度=" + strconv.Itoa(len(datagramBytes)))
	}

	// 4. 提取数据部分
	var data []byte // 如果 head.Data_length 为 0，则 data 保持为 nil
	if head.Data_length > 0 {
		data = datagramBytes[protocol.DATAGRAM_MIN_LEN:]
	}

	// 5. 获取除去校验码的帧头的序列化值
	headSerialized, err := head.SerializeWithoutCheckCode()
	if err != nil {
		return nil, errors.New("序列化头部失败: " + err.Error())
	}

	// 6. 计算校验码
	hasher := xxhash.New()
	hasher.Write(headSerialized)
	if len(data) > 0 {
		hasher.Write(data) // 数据部分
	}
	calculatedCheckCode := uint32(hasher.Sum64())

	// 7. 校验码匹配
	if calculatedCheckCode != head.Check_code {
		return nil, errors.New("校验码不匹配：计算得到的校验码=" + strconv.FormatUint(uint64(calculatedCheckCode), 10) +
			"，报文中的校验码=" + strconv.FormatUint(uint64(head.Check_code), 10))
	}

	return &Datagram{
		Head: head,
		Data: data,
	}, nil
}
