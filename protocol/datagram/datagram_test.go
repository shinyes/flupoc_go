package datagram

import (
	"testing"

	"github.com/cykyes/flupoc-go/protocol/head"
)

func TestSerializeSyncsDataLengthWithPayload(t *testing.T) {
	dg := &Datagram{
		Head: &head.Header{
			Protocol:   head.ProtocolID,
			Type:       head.MsgRequest,
			ChannelID:  7,
			DataLength: 99, // 故意制造与 Data 不一致
		},
		Data: []byte("abc"),
	}

	raw := dg.Serialize()

	if dg.Head.DataLength != 3 {
		t.Fatalf("期望 Head.DataLength 被同步为 3，实际为 %d", dg.Head.DataLength)
	}

	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if parsed.Head.DataLength != 3 {
		t.Fatalf("期望序列化后的头长度为 3，实际为 %d", parsed.Head.DataLength)
	}
	if string(parsed.Data) != "abc" {
		t.Fatalf("期望 payload=abc，实际=%q", string(parsed.Data))
	}
}

func TestSerializeWithNilHeadUsesDefaultProtocol(t *testing.T) {
	dg := &Datagram{Data: nil}

	raw := dg.Serialize()
	parsed, err := Parse(raw)
	if err != nil {
		t.Fatalf("反序列化失败: %v", err)
	}

	if parsed.Head.Protocol != head.ProtocolID {
		t.Fatalf("期望协议号为 0x%02X，实际为 0x%02X", head.ProtocolID, parsed.Head.Protocol)
	}
	if parsed.Head.DataLength != 0 {
		t.Fatalf("期望数据长度为 0，实际为 %d", parsed.Head.DataLength)
	}
}
