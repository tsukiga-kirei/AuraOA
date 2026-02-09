package observability

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const TraceIDHeader = "X-Trace-ID"

// TraceMiddleware injects a Trace ID into every request.
func TraceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		traceID := c.GetHeader(TraceIDHeader)
		if traceID == "" {
			traceID = uuid.New().String()
		}
		c.Set("trace_id", traceID)
		c.Header(TraceIDHeader, traceID)
		c.Next()
	}
}

// MetricsMiddleware records API call success/failure.
func MetricsMiddleware(m *Metrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)

		success := c.Writer.Status() < 500
		m.RecordAPICall(success)

		// If this was an audit call, also record model response time
		if c.GetFloat64("model_response_ms") > 0 {
			m.RecordModelResponse(int64(c.GetFloat64("model_response_ms")))
		}

		_ = duration // available for logging if needed
	}
}
