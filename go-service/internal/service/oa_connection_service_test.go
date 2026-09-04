package service

import (
	"testing"
)

func TestBuildOAProcessURL(t *testing.T) {
	tests := []struct {
		name      string
		baseURL   string
		template  string
		oaType    string
		processID string
		want      string
	}{
		{
			name:      "空 processID 返回空",
			baseURL:   "http://oa.example.com",
			template:  "",
			oaType:    "weaver_e9",
			processID: "",
			want:      "",
		},
		{
			name:      "未配置模板-泛微E9默认规则",
			baseURL:   "http://oa.example.com",
			template:  "",
			oaType:    "weaver_e9",
			processID: "613446",
			want:      "http://oa.example.com/workflow/request/ViewRequestForwardSPA.jsp?requestid=613446",
		},
		{
			name:      "未配置模板-带尾部斜杠自动去除",
			baseURL:   "http://oa.example.com:8088/",
			template:  "",
			oaType:    "weaver_e9",
			processID: "12345",
			want:      "http://oa.example.com:8088/workflow/request/ViewRequestForwardSPA.jsp?requestid=12345",
		},
		{
			name:      "未配置协议自动补全http",
			baseURL:   "oa.example.com",
			template:  "",
			oaType:    "weaver_e9",
			processID: "888",
			want:      "http://oa.example.com/workflow/request/ViewRequestForwardSPA.jsp?requestid=888",
		},
		{
			name:      "自定义相对路径模板-{process_id}",
			baseURL:   "https://oa.example.com",
			template:  "/custom/process/view.jsp?id={process_id}",
			oaType:    "other_oa",
			processID: "999",
			want:      "https://oa.example.com/custom/process/view.jsp?id=999",
		},
		{
			name:      "自定义相对路径模板-{requestid}",
			baseURL:   "https://oa.example.com",
			template:  "/workflow/detail?req={requestid}",
			oaType:    "weaver_e9",
			processID: "777",
			want:      "https://oa.example.com/workflow/detail?req=777",
		},
		{
			name:      "自定义绝对路径模板直接覆盖域名",
			baseURL:   "http://default.com",
			template:  "https://standalone-oa.corp.com/flow/{process_id}",
			oaType:    "weaver_e9",
			processID: "555",
			want:      "https://standalone-oa.corp.com/flow/555",
		},
		{
			name:      "两处均为空返回空",
			baseURL:   "",
			template:  "",
			oaType:    "weaver_e9",
			processID: "100",
			want:      "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildOAProcessURL(tt.baseURL, tt.template, tt.oaType, tt.processID)
			if got != tt.want {
				t.Errorf("BuildOAProcessURL() = %q, want %q", got, tt.want)
			}
		})
	}
}
