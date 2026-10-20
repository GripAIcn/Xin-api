package inbound

import (
	"strings"
	"testing"
)

// TestCodecKeepaliveEvent 各入站协议的保活事件形态：SSE 注释（chat/responses/gemini）与 Anthropic 原生 ping
func TestCodecKeepaliveEvent(t *testing.T) {
	cases := []struct {
		protocol string
		want     string
	}{
		{ProtocolChat, ": ping\n\n"},
		{ProtocolResponses, ": ping\n\n"},
		{ProtocolGemini, ": ping\n\n"},
		{ProtocolMessages, "event: ping\ndata: {\"type\": \"ping\"}\n\n"},
	}
	registry := NewRegistry()
	for _, tc := range cases {
		codec := registry.New(tc.protocol)
		got := string(codec.KeepaliveEvent())
		if got != tc.want {
			t.Fatalf("%s 协议保活事件 = %q, 期望 %q", tc.protocol, got, tc.want)
		}
	}
}

// TestKeepaliveEventParsableAsSSE 保活事件必须是可解析的 SSE 块（以空行结束）
func TestKeepaliveEventParsableAsSSE(t *testing.T) {
	registry := NewRegistry()
	for _, protocol := range []string{ProtocolChat, ProtocolResponses, ProtocolGemini, ProtocolMessages} {
		codec := registry.New(protocol)
		event := string(codec.KeepaliveEvent())
		if !strings.HasSuffix(event, "\n\n") {
			t.Fatalf("%s 保活事件未以空行结尾: %q", protocol, event)
		}
	}
}
