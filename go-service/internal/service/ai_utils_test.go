// 自定义标识和业务数据均为虚拟样例；厂商标准表结构按适配器约定保留。
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

func TestFormatAttachmentsWithDetailTableAndRow(t *testing.T) {
	attachments := []oa.AttachmentInfo{
		{
			FileName:  "主表附件.pdf",
			FieldKey:  "T006",
			FieldName: "附件上传",
			Content:   "主表内容",
		},
		{
			FileName:    "发票1.pdf",
			FieldKey:    "fpfj",
			FieldName:   "发票附件",
			DetailTable: "formtable_main_900003_dt1",
			RowIndex:    1,
			Content:     "发票1内容",
		},
		{
			FileName:    "发票2.pdf",
			FieldKey:    "fpfj",
			FieldName:   "发票附件",
			DetailTable: "formtable_main_900003_dt1",
			RowIndex:    2,
			Content:     "发票2内容",
		},
	}
	got := formatAttachments(attachments, 8000)
	if !strings.Contains(got, "### 附件字段：附件上传（T006）") {
		t.Errorf("未能正确格式化主表附件头: %q", got)
	}
	if !strings.Contains(got, "### 附件字段：发票附件（fpfj - 明细表1 第 1 行）") {
		t.Errorf("未能正确格式化明细表第1行附件头: %q", got)
	}
	if !strings.Contains(got, "### 附件字段：发票附件（fpfj - 明细表1 第 2 行）") {
		t.Errorf("未能正确格式化明细表第2行附件头: %q", got)
	}
}

func TestFormatAttachments_WithZipArchive(t *testing.T) {
	attachments := []oa.AttachmentInfo{
		{
			FileName:  "独立通知.txt",
			FieldKey:  "fj",
			FieldName: "流程附件",
			Content:   "独立通知正文",
		},
		{
			FileName:        "方案.docx",
			FieldKey:        "fj",
			FieldName:       "流程附件",
			Content:         "方案详细内容",
			ArchiveFileName: "资料包.zip",
		},
		{
			FileName:        "图纸.dwg",
			FieldKey:        "fj",
			FieldName:       "流程附件",
			Error:           "文件类型 dwg 不在支持列表中",
			ArchiveFileName: "资料包.zip",
		},
	}

	got := formatAttachments(attachments, 8000)
	if !strings.Contains(got, "- 附件 1：独立通知.txt") {
		t.Errorf("未能正确展示独立文件: %s", got)
	}
	if !strings.Contains(got, "- 附件 2：资料包.zip（ZIP 压缩包，共解压出 2 个文件）") {
		t.Errorf("未能正确展示压缩包汇总项: %s", got)
	}
	if !strings.Contains(got, "* [已提取内容] 1. 方案.docx") {
		t.Errorf("未能正确标记已提取内容子文件: %s", got)
	}
	if !strings.Contains(got, "* [未提取内容] 2. 图纸.dwg") {
		t.Errorf("未能正确标记未提取内容子文件: %s", got)
	}
	if !strings.Contains(got, "说明：文件类型 dwg 不在支持列表中") {
		t.Errorf("未能正确展示未提取说明: %s", got)
	}
}
