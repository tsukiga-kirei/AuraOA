package validate

import (
	"strings"
	"testing"
)

func TestIsLoginUsername(t *testing.T) {
	t.Parallel()

	valid := []string{"a", "lism", "3019325", "user_01", "_bot", "A1"}
	for _, username := range valid {
		if !IsLoginUsername(username) {
			t.Fatalf("期望通过: %q", username)
		}
	}

	invalid := []string{"", "李XX", "user-01", "user.01", " user", "a/b", strings.Repeat("a", 101)}
	for _, username := range invalid {
		if IsLoginUsername(username) {
			t.Fatalf("期望拒绝: %q", username)
		}
	}
}
