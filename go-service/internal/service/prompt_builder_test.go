package service

import "testing"

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
