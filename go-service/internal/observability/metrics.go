package observability

import (
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Metrics tracks API success rates and model response times.
type Metrics struct {
	mu              sync.RWMutex
	apiCalls        int64
	apiSuccesses    int64
	modelTotalMs    int64
	modelCallCount  int64
	alerts          []Alert
	alertThresholds AlertThresholds
}

type AlertThresholds struct {
	MaxModelResponseMs int64   // alert if avg model response exceeds this
	MinAPISuccessRate  float64 // alert if success rate drops below this (0-1)
}

type Alert struct {
	ID         string    `json:"id"`
	MetricName string    `json:"metric_name"`
	Value      float64   `json:"value"`
	Threshold  float64   `json:"threshold"`
	CreatedAt  time.Time `json:"created_at"`
}

func NewMetrics(thresholds AlertThresholds) *Metrics {
	if thresholds.MaxModelResponseMs == 0 {
		thresholds.MaxModelResponseMs = 5000
	}
	if thresholds.MinAPISuccessRate == 0 {
		thresholds.MinAPISuccessRate = 0.95
	}
	return &Metrics{
		alertThresholds: thresholds,
	}
}

// RecordAPICall records an API call result.
func (m *Metrics) RecordAPICall(success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.apiCalls++
	if success {
		m.apiSuccesses++
	}
	m.checkAPISuccessRate()
}

// RecordModelResponse records a model response time.
func (m *Metrics) RecordModelResponse(durationMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.modelTotalMs += durationMs
	m.modelCallCount++
	m.checkModelResponseTime()
}

// GetAPISuccessRate returns the current API success rate.
func (m *Metrics) GetAPISuccessRate() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.apiCalls == 0 {
		return 1.0
	}
	return float64(m.apiSuccesses) / float64(m.apiCalls)
}

// GetAvgModelResponseMs returns the average model response time in ms.
func (m *Metrics) GetAvgModelResponseMs() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.modelCallCount == 0 {
		return 0
	}
	return m.modelTotalMs / m.modelCallCount
}

// GetAlerts returns all triggered alerts.
func (m *Metrics) GetAlerts() []Alert {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]Alert, len(m.alerts))
	copy(result, m.alerts)
	return result
}

func (m *Metrics) checkAPISuccessRate() {
	if m.apiCalls < 10 {
		return // not enough data
	}
	rate := float64(m.apiSuccesses) / float64(m.apiCalls)
	if rate < m.alertThresholds.MinAPISuccessRate {
		alert := Alert{
			ID:         uuid.New().String(),
			MetricName: "api_success_rate",
			Value:      rate,
			Threshold:  m.alertThresholds.MinAPISuccessRate,
			CreatedAt:  time.Now(),
		}
		m.alerts = append(m.alerts, alert)
		log.Printf("[ALERT] API success rate %.2f%% below threshold %.2f%%", rate*100, m.alertThresholds.MinAPISuccessRate*100)
	}
}

func (m *Metrics) checkModelResponseTime() {
	if m.modelCallCount < 5 {
		return
	}
	avg := m.modelTotalMs / m.modelCallCount
	if avg > m.alertThresholds.MaxModelResponseMs {
		alert := Alert{
			ID:         uuid.New().String(),
			MetricName: "model_response_time",
			Value:      float64(avg),
			Threshold:  float64(m.alertThresholds.MaxModelResponseMs),
			CreatedAt:  time.Now(),
		}
		m.alerts = append(m.alerts, alert)
		log.Printf("[ALERT] Avg model response %dms exceeds threshold %dms", avg, m.alertThresholds.MaxModelResponseMs)
	}
}

// GenerateTraceID creates a unique trace ID for cross-service tracking.
func GenerateTraceID() string {
	return uuid.New().String()
}
