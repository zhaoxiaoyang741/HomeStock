package feishu

import (
	"testing"

	larkim "github.com/larksuite/oapi-sdk-go/v3/service/im/v1"
)

func TestExtractContent(t *testing.T) {
	tests := []struct {
		name        string
		messageType string
		rawContent  string
		want        string
	}{
		{
			name:        "text message",
			messageType: "text",
			rawContent:  `{"text": "hello world"}`,
			want:        "hello world",
		},
		{
			name:        "text message invalid JSON",
			messageType: "text",
			rawContent:  `not json`,
			want:        "not json",
		},
		{
			name:        "image message returns empty",
			messageType: "image",
			rawContent:  `{"image_key": "img_xxx"}`,
			want:        "",
		},
		{
			name:        "file message with filename",
			messageType: "file",
			rawContent:  `{"file_key": "file_xxx", "file_name": "report.pdf"}`,
			want:        "report.pdf",
		},
		{
			name:        "empty raw content",
			messageType: "text",
			rawContent:  "",
			want:        "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractContent(tt.messageType, tt.rawContent)
			if got != tt.want {
				t.Errorf("extractContent(%q, %q) = %q, want %q", tt.messageType, tt.rawContent, got, tt.want)
			}
		})
	}
}

func TestExtractFeishuSenderID(t *testing.T) {
	strPtr := func(s string) *string { return &s }

	tests := []struct {
		name   string
		sender *larkim.EventSender
		want   string
	}{
		{
			name:   "nil sender",
			sender: nil,
			want:   "",
		},
		{
			name:   "nil sender ID",
			sender: &larkim.EventSender{SenderId: nil},
			want:   "",
		},
		{
			name: "userId preferred",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr("u_abc123"),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "u_abc123",
		},
		{
			name: "openId fallback",
			sender: &larkim.EventSender{
				SenderId: &larkim.UserId{
					UserId:  strPtr(""),
					OpenId:  strPtr("ou_def456"),
					UnionId: strPtr("on_ghi789"),
				},
			},
			want: "ou_def456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractFeishuSenderID(tt.sender)
			if got != tt.want {
				t.Errorf("extractFeishuSenderID() = %q, want %q", got, tt.want)
			}
		})
	}
}
