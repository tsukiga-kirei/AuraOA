package service

import (
	"encoding/json"
	"testing"
)

func TestExtractMinerUMarkdown(t *testing.T) {
	tests := []struct {
		name    string
		payload string
		want    string
	}{
		{
			name:    "document result",
			payload: `{"document":{"md_content":"from document"}}`,
			want:    "from document",
		},
		{
			name:    "files result",
			payload: `{"files":{"md_content":"from files"}}`,
			want:    "from files",
		},
		{
			name:    "filename keyed result",
			payload: `{"合同.pdf":{"md_content":"from filename"}}`,
			want:    "from filename",
		},
		{
			name:    "direct markdown result",
			payload: `{"md_content":"from direct"}`,
			want:    "from direct",
		},
		{
			name:    "empty result",
			payload: `{}`,
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractMinerUMarkdown(json.RawMessage(tt.payload))
			if got != tt.want {
				t.Fatalf("extractMinerUMarkdown() = %q, want %q", got, tt.want)
			}
		})
	}
}
