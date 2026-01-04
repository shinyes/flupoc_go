package datagram

import (
	"testing"

	protocol "github.com/cykyes/flupoc-go/protocol/head"
)

// 测试 NewDatagram 函数
func TestNewDatagram(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint16
		msgType   uint8
		data      []byte
		wantErr   bool
	}{
		{"正常创建无数据", 1, protocol.MSG_TYPE_PING, nil, false},
		{"正常创建有数据", 2, protocol.MSG_TYPE_PONG, []byte("hello"), false},
		{"正常创建长数据", 3, protocol.MSG_TYPE_REQ, []byte("this is a longer test data for datagram"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewDatagram(tt.channelID, tt.msgType, tt.data)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewDatagram() 错误 = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			if got.Head.Protocol_id != protocol.FP_PROTOCOL_ID {
				t.Errorf("NewDatagram() Protocol_id = %v, want %v", got.Head.Protocol_id, protocol.FP_PROTOCOL_ID)
			}
			if got.Head.Channel_id != tt.channelID {
				t.Errorf("NewDatagram() Channel_id = %v, want %v", got.Head.Channel_id, tt.channelID)
			}
			if got.Head.Msg_type != tt.msgType {
				t.Errorf("NewDatagram() Msg_type = %v, want %v", got.Head.Msg_type, tt.msgType)
			}
			if got.Head.Data_length != uint32(len(tt.data)) {
				t.Errorf("NewDatagram() Data_length = %v, want %v", got.Head.Data_length, uint32(len(tt.data)))
			}

			// 修复切片比较问题
			if !equalBytes(got.Data, tt.data) {
				t.Errorf("NewDatagram() Data = %v, want %v", got.Data, tt.data)
			}
			if got.Head.Check_code == 0 {
				t.Errorf("NewDatagram() Check_code should not be 0")
			}
		})
	}
}

// 测试 Serialize 函数
func TestDatagram_Serialize(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint16
		msgType   uint8
		data      []byte
		wantErr   bool
	}{
		{"序列化无数据", 1, protocol.MSG_TYPE_PING, nil, false},
		{"序列化有数据", 2, protocol.MSG_TYPE_PONG, []byte("hello"), false},
		{"序列化长数据", 3, protocol.MSG_TYPE_REQ, []byte("this is a longer test data for datagram"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			datagram, err := NewDatagram(tt.channelID, tt.msgType, tt.data)
			if err != nil {
				t.Fatalf("NewDatagram() 错误 = %v", err)
			}

			got, err := datagram.Serialize()
			if (err != nil) != tt.wantErr {
				t.Errorf("Datagram.Serialize() 错误 = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// 检查序列化后数据的长度是否正确
			expectedLen := protocol.DATAGRAM_MIN_LEN + len(tt.data)
			if len(got) != expectedLen {
				t.Errorf("Datagram.Serialize() length = %v, want %v", len(got), expectedLen)
			}
		})
	}
}

// 测试 Deserialize 函数
func TestDatagram_Deserialize(t *testing.T) {
	tests := []struct {
		name      string
		channelID uint16
		msgType   uint8
		data      []byte
		wantErr   bool
	}{
		{"反序列化无数据", 1, protocol.MSG_TYPE_PING, nil, false},
		{"反序列化有数据", 2, protocol.MSG_TYPE_PONG, []byte("hello"), false},
		{"反序列化长数据", 3, protocol.MSG_TYPE_REQ, []byte("this is a longer test data for datagram"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 先创建一个datagram并序列化
			original, err := NewDatagram(tt.channelID, tt.msgType, tt.data)
			if err != nil {
				t.Fatalf("NewDatagram() 错误 = %v", err)
			}

			serialized, err := original.Serialize()
			if err != nil {
				t.Fatalf("Datagram.Serialize() 错误 = %v", err)
			}

			// 然后反序列化
			got, err := Deserialize(serialized)
			if (err != nil) != tt.wantErr {
				t.Errorf("Deserialize() 错误 = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}

			// 比较反序列化后的datagram与原始datagram
			if got.Head.Protocol_id != original.Head.Protocol_id {
				t.Errorf("Deserialize() Protocol_id = %v, want %v", got.Head.Protocol_id, original.Head.Protocol_id)
			}
			if got.Head.Channel_id != original.Head.Channel_id {
				t.Errorf("Deserialize() Channel_id = %v, want %v", got.Head.Channel_id, original.Head.Channel_id)
			}
			if got.Head.Msg_type != original.Head.Msg_type {
				t.Errorf("Deserialize() Msg_type = %v, want %v", got.Head.Msg_type, original.Head.Msg_type)
			}
			if got.Head.Check_code != original.Head.Check_code {
				t.Errorf("Deserialize() Check_code = %v, want %v", got.Head.Check_code, original.Head.Check_code)
			}
			if got.Head.Data_length != original.Head.Data_length {
				t.Errorf("Deserialize() Data_length = %v, want %v", got.Head.Data_length, original.Head.Data_length)
			}
			// 修复切片比较问题
			if !equalBytes(got.Data, original.Data) {
				t.Errorf("Deserialize() Data = %v, want %v", got.Data, original.Data)
			}
		})
	}
}

// 测试序列化和反序列化结合
func TestDatagram_SerializeDeserialize(t *testing.T) {
	testCases := []struct {
		name      string
		channelID uint16
		msgType   uint8
		data      []byte
	}{
		{"PING消息无数据", 1, protocol.MSG_TYPE_PING, nil},
		{"PONG消息有数据", 2, protocol.MSG_TYPE_PONG, []byte("pong data")},
		{"REQ消息有数据", 3, protocol.MSG_TYPE_REQ, []byte("request data")},
		{"RES消息有数据", 4, protocol.MSG_TYPE_RES, []byte("response data")},
		{"创建频道消息", 5, protocol.MSG_TYPE_CREATE_CHANNEL, []byte("create channel data")},
		{"关闭频道消息", 6, protocol.MSG_TYPE_CLOSE_CHANNEL, []byte("close channel data")},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// 创建原始数据报
			original, err := NewDatagram(tc.channelID, tc.msgType, tc.data)
			if err != nil {
				t.Fatalf("NewDatagram 失败: %v", err)
			}

			// 序列化
			serialized, err := original.Serialize()
			if err != nil {
				t.Fatalf("Serialize 失败: %v", err)
			}

			// 反序列化
			deserialized, err := Deserialize(serialized)
			if err != nil {
				t.Fatalf("Deserialize 失败: %v", err)
			}

			// 验证反序列化后的数据报与原始数据报一致
			if deserialized.Head.Protocol_id != original.Head.Protocol_id {
				t.Errorf("Protocol_id 不匹配: got %d, want %d", deserialized.Head.Protocol_id, original.Head.Protocol_id)
			}
			if deserialized.Head.Msg_type != original.Head.Msg_type {
				t.Errorf("Msg_type 不匹配: got %d, want %d", deserialized.Head.Msg_type, original.Head.Msg_type)
			}
			if deserialized.Head.Channel_id != original.Head.Channel_id {
				t.Errorf("Channel_id 不匹配: got %d, want %d", deserialized.Head.Channel_id, original.Head.Channel_id)
			}
			if deserialized.Head.Check_code != original.Head.Check_code {
				t.Errorf("Check_code 不匹配: got %d, want %d", deserialized.Head.Check_code, original.Head.Check_code)
			}
			if deserialized.Head.Data_length != original.Head.Data_length {
				t.Errorf("Data_length 不匹配: got %d, want %d", deserialized.Head.Data_length, original.Head.Data_length)
			}
			// 修复切片比较问题
			if !equalBytes(deserialized.Data, original.Data) {
				t.Errorf("Data 不匹配: got %v, want %v", deserialized.Data, original.Data)
			}
		})
	}
}

// 测试错误情况
func TestDatagram_Deserialize_Errors(t *testing.T) {
	t.Run("反序列化长度不足", func(t *testing.T) {
		shortData := make([]byte, protocol.DATAGRAM_MIN_LEN-1)
		_, err := Deserialize(shortData)
		if err == nil {
			t.Error("Deserialize 应该对长度不足返回错误")
		}
	})

	t.Run("反序列化数据长度不匹配", func(t *testing.T) {
		// 创建一个正常的datagram
		original, err := NewDatagram(1, protocol.MSG_TYPE_PING, []byte("test"))
		if err != nil {
			t.Fatalf("NewDatagram 失败: %v", err)
		}

		serialized, err := original.Serialize()
		if err != nil {
			t.Fatalf("Serialize 失败: %v", err)
		}

		// 截断数据部分，使长度不匹配
		truncated := serialized[:protocol.DATAGRAM_MIN_LEN+2] // 只保留部分数据
		_, err = Deserialize(truncated)
		if err == nil {
			t.Error("Deserialize 应该对长度不匹配返回错误")
		}
	})
}

// 辅助函数，用于比较两个字节切片是否相等
func equalBytes(a, b []byte) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
