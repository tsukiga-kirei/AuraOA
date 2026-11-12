package csvsafe

import "testing"

func TestCell(t *testing.T) {
	cases := []struct {
		name  string
		input string
		want  string
	}{
		{name: "普通文本", input: "报销申请", want: "报销申请"},
		{name: "空值", input: "", want: ""},
		{name: "等号公式", input: "=SUM(A1:A3)", want: "'=SUM(A1:A3)"},
		{name: "加号公式", input: "+1+1", want: "'+1+1"},
		{name: "减号公式", input: "-2+3", want: "'-2+3"},
		{name: "at 公式", input: "@SUM(A1)", want: "'@SUM(A1)"},
		{name: "制表符", input: "\tcmd", want: "'\tcmd"},
		{name: "回车", input: "\rcmd", want: "'\rcmd"},
		{name: "中间等号不受影响", input: "金额=100", want: "金额=100"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := Cell(tc.input); got != tc.want {
				t.Fatalf("Cell(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}
