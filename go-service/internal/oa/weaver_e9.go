package oa

import (
	"context"
	"fmt"
	"log"
	"time"
)

// WeaverE9Config holds connection config for Weaver E9 OA.
type WeaverE9Config struct {
	JDBCUrl       string
	MaxRetries    int
	RetryInterval time.Duration
}

// WeaverE9Adapter implements OAAdapter for Weaver E9.
type WeaverE9Adapter struct {
	config    WeaverE9Config
	connected bool
}

func NewWeaverE9Adapter(config WeaverE9Config) *WeaverE9Adapter {
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryInterval == 0 {
		config.RetryInterval = 5 * time.Second
	}
	return &WeaverE9Adapter{config: config}
}

func (a *WeaverE9Adapter) FetchTodoProcesses(ctx context.Context, userID string) ([]OAProcess, error) {
	if err := a.ensureConnected(ctx); err != nil {
		return nil, err
	}
	// TODO: actual JDBC query against Weaver E9 tables
	// Placeholder: return empty list
	return []OAProcess{}, nil
}

func (a *WeaverE9Adapter) FetchProcessDetail(ctx context.Context, processID string) (ProcessFormData, error) {
	if err := a.ensureConnected(ctx); err != nil {
		return ProcessFormData{}, err
	}
	// TODO: actual JDBC query to map E9 form fields to ProcessFormData
	return ProcessFormData{
		ProcessID:  processID,
		FormFields: []FormField{},
	}, nil
}

func (a *WeaverE9Adapter) HealthCheck(ctx context.Context) error {
	// TODO: actual connection ping
	return nil
}

// ensureConnected attempts to connect with retry logic.
func (a *WeaverE9Adapter) ensureConnected(ctx context.Context) error {
	if a.connected {
		return nil
	}
	for i := 0; i < a.config.MaxRetries; i++ {
		err := a.connect(ctx)
		if err == nil {
			a.connected = true
			return nil
		}
		log.Printf("OA connection attempt %d/%d failed: %v", i+1, a.config.MaxRetries, err)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(a.config.RetryInterval):
		}
	}
	return fmt.Errorf("failed to connect to OA after %d retries", a.config.MaxRetries)
}

func (a *WeaverE9Adapter) connect(ctx context.Context) error {
	// TODO: establish actual JDBC connection
	// For now, simulate success
	return nil
}

// ProcessFilter supports filtering processes by directory, path, or ID.
type ProcessFilter struct {
	Directory   string `json:"directory,omitempty"`
	Path        string `json:"path,omitempty"`
	ProcessID   string `json:"process_id,omitempty"`
}

// FilterProcesses filters a list of OA processes by the given criteria.
func FilterProcesses(processes []OAProcess, filter ProcessFilter) []OAProcess {
	if filter.ProcessID == "" && filter.Directory == "" && filter.Path == "" {
		return processes
	}

	var result []OAProcess
	for _, p := range processes {
		if filter.ProcessID != "" && p.ProcessID != filter.ProcessID {
			continue
		}
		// Directory and path filtering would match against process metadata
		// TODO: implement when E9 table structure is mapped
		result = append(result, p)
	}
	return result
}
