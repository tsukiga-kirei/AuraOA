package service

import (
	"strings"
	"testing"
	"unicode/utf8"

	"auraoa/go-service/internal/pkg/oa"
)

func TestCleanJSONResponse_markdownFences(t *testing.T) {
	want := `{"recommendation":"approve","overall_score":90}`
	cases := []string{
		"```json\n" + want + "\n```",
		"```json\r\n" + want + "\r\n```",
		"```JSON\n" + want + "\n```",
		"说明如下：\n```json\n" + want + "\n```\n谢谢",
		"```\n" + want + "\n```",
		"... ```\n" + want + "\n```",
	}
	for _, raw := range cases {
		got := cleanJSONResponse(raw)
		if got != want {
			t.Fatalf("cleanJSONResponse(%q)\n got %q\n want %q", raw, got, want)
		}
	}
}

func TestExtractJSONFromMarkdownFence_noFence(t *testing.T) {
	s := `{"a":1}`
	if extractJSONFromMarkdownFence(s) != s {
		t.Fatalf("expected unchanged, got %q", extractJSONFromMarkdownFence(s))
	}
}

func TestTruncateUTF8BytesDoesNotSplitChineseRune(t *testing.T) {
	got := truncateUTF8Bytes("中文附件正文", 4)
	if !utf8.ValidString(got) {
		t.Fatalf("truncateUTF8Bytes() returned invalid UTF-8: %q", got)
	}
	if got != "中..." {
		t.Fatalf("truncateUTF8Bytes() = %q, want %q", got, "中...")
	}
}

func TestFormatAttachmentsUsesConfiguredByteLimit(t *testing.T) {
	got := formatAttachments([]oa.AttachmentInfo{{
		FileName:         "通知.doc",
		FieldKey:         "fj",
		FieldName:        "附件",
		Content:          "中文附件正文",
		ContentLimitMode: attachmentAIContentLimitBytes,
		ContentMaxBytes:  4,
	}}, 8000)
	if !utf8.ValidString(got) {
		t.Fatalf("formatAttachments() returned invalid UTF-8: %q", got)
	}
	if !strings.Contains(got, "中...") || strings.Contains(got, "文附件正文") {
		t.Fatalf("formatAttachments() did not apply configured byte limit: %q", got)
	}
}

func TestFormatAttachmentsUnlimitedKeepsAllContent(t *testing.T) {
	content := strings.Repeat("完整正文", 20)
	got := formatAttachments([]oa.AttachmentInfo{{
		FileName:         "通知.ofd",
		FieldKey:         "fj",
		FieldName:        "附件",
		Content:          content,
		ContentLimitMode: attachmentAIContentLimitUnlimited,
		ContentMaxBytes:  3,
	}}, 3)
	if !strings.Contains(got, content) {
		t.Fatalf("formatAttachments() truncated unlimited content: %q", got)
	}
}
