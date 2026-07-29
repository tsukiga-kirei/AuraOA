package middleware

import "testing"

func TestIsLowValuePollingPath_EmbedJobs(t *testing.T) {
	t.Parallel()

	paths := []string{
		"/api/embed/jobs/audit-job-id",
		"/api/embed/summary/jobs/summary-job-id",
	}
	for _, path := range paths {
		if !IsLowValuePollingPath(path) {
			t.Errorf("嵌入任务轮询路径应归类为低价值轮询: %s", path)
		}
	}
}
