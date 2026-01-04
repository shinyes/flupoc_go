package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
)

const (
	FP_PROTOCOL_ID uint8 = 0xCF

	MSG_TYPE_PING           uint8 = 0x01 // 心跳请求
	MSG_TYPE_PONG           uint8 = 0x02 // 心跳响应
	MSG_TYPE_REQ            uint8 = 0x03 // 属于请求报文，对于请求报文会创建一个频道，待服务端响应之后，这个频道就会关闭
	MSG_TYPE_RES            uint8 = 0x04
	MSG_TYPE_CREATE_CHANNEL uint8 = 0x05 // 创建一个持久频道，id 由帧头中的 Protocol_id指定
	MSG_TYPE_CLOSE_CHANNEL  uint8 = 0x06 // 关闭一个持久通道，id 由帧头中的 Protocol_id指定

	DATAGRAM_MIN_LEN int = 12 // 整个报文最小值为 12 bytes，此时数据部分的长度为 0
	CHECK_CODE_LEN   int = 32 // 使用 XXSH32 算法（实际上是64位哈希值的前32位）
)

type Head struct {
	Protocol_id uint8
	Msg_type    uint8
	Channel_id  uint16
	Check_code  uint32 // 新增校验码字段
	Data_length uint32 // 数据部分的字节长度
}

// 序列化帧头不包含校验码的部分
func (h *Head) SerializeWithoutCheckCode() ([]byte, error) {
	// 强制将Protocol_id设置为常量PROTOCOL_ID
	h.Protocol_id = FP_PROTOCOL_ID

	tempBuf := bytes.NewBuffer(make([]byte, 0, 8))
	if err := binary.Write(tempBuf, binary.BigEndian, h.Protocol_id); err != nil {
		return nil, errors.New("序列化Protocol_id失败: " + err.Error())
	}
	if err := binary.Write(tempBuf, binary.BigEndian, h.Msg_type); err != nil {
		return nil, errors.New("序列化Msg_type失败: " + err.Error())
	}
	if err := binary.Write(tempBuf, binary.BigEndian, h.Channel_id); err != nil {
		return nil, errors.New("序列化Channel_id失败: " + err.Error())
	}
	if err := binary.Write(tempBuf, binary.BigEndian, h.Data_length); err != nil {
		return nil, errors.New("序列化Length失败: " + err.Error())
	}
	return tempBuf.Bytes(), nil
}

// 反序列化帧头
func Deserialize(headBytes []byte) (*Head, error) {
	buf := bytes.NewReader(headBytes)
	head := &Head{}
	if err := binary.Read(buf, binary.BigEndian, &head.Protocol_id); err != nil {
		return nil, errors.New("反序列化Protocol_id失败: " + err.Error())
	}
	if err := binary.Read(buf, binary.BigEndian, &head.Msg_type); err != nil {
		return nil, errors.New("反序列化Msg_type失败: " + err.Error())
	}
	if err := binary.Read(buf, binary.BigEndian, &head.Channel_id); err != nil {
		return nil, errors.New("反序列化Channel_id失败: " + err.Error())
	}
	if err := binary.Read(buf, binary.BigEndian, &head.Check_code); err != nil {
		return nil, errors.New("反序列化Check_code失败: " + err.Error())
	}
	if err := binary.Read(buf, binary.BigEndian, &head.Data_length); err != nil {
		return nil, errors.New("反序列化Length失败: " + err.Error())
	}
	return head, nil
}
