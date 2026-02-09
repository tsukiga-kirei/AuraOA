package rule

import "context"

type RuleScope string

const (
	RuleScopeMandatory  RuleScope = "mandatory"
	RuleScopeDefaultOn  RuleScope = "default_on"
	RuleScopeDefaultOff RuleScope = "default_off"
)

type RuleSource string

const (
	RuleSourceTenant RuleSource = "tenant"
	RuleSourceUser   RuleSource = "user"
)

type MergedRule struct {
	ID       string     `json:"id"`
	Content  string     `json:"content"`
	Scope    RuleScope  `json:"scope"`
	Source   RuleSource `json:"source"`
	IsLocked bool       `json:"is_locked"`
	Priority int        `json:"priority"`
}

type ConfigurableRule struct {
	ID      string    `json:"id"`
	Content string    `json:"content"`
	Scope   RuleScope `json:"scope"`
	Enabled bool      `json:"enabled"`
}

// RuleEngine handles rule loading, merging and priority resolution.
type RuleEngine interface {
	// MergeRules merges rules by priority: tenant mandatory > user private > tenant default
	MergeRules(ctx context.Context, tenantID string, userID string, processType string) ([]MergedRule, error)
	// GetConfigurableRules returns rules that the user can toggle on/off
	GetConfigurableRules(ctx context.Context, tenantID string, userID string) ([]ConfigurableRule, error)
}
