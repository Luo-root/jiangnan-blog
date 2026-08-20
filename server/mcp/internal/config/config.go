package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config 是一份 config.yaml。启动必填只有 schema.visibility_policy 与 admin_auth。
type Config struct {
	Server    ServerConfig    `yaml:"server"`
	Admin     AdminConfig     `yaml:"admin"`
	AdminAuth AdminAuthConfig `yaml:"admin_auth"`
	Auth      AuthConfig      `yaml:"auth"`
	Vault     VaultConfig     `yaml:"vault"`
	Workbase  WorkbaseConfig  `yaml:"workbase"`
	Inbox     InboxConfig     `yaml:"inbox"`
	Index     IndexConfig     `yaml:"index"`
	Knowledge KnowledgeConfig `yaml:"knowledge"`
	Audit     AuditConfig     `yaml:"audit"`
	Schema    Schema          `yaml:"schema"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type AdminConfig struct {
	Listen         string `yaml:"listen"`
	SessionTTL     int    `yaml:"session_ttl"`
	LoginRateLimit int    `yaml:"login_rate_limit"`
}

type AdminAuthConfig struct {
	User     string `yaml:"user"`
	PassHash string `yaml:"pass_hash"`
}

type AuthConfig struct {
	GracePeriodHours int `yaml:"grace_period_hours"`
}

type VaultConfig struct {
	Root   string `yaml:"root"`
	GitDir string `yaml:"git_dir"`
}

type WorkbaseConfig struct {
	Root       string `yaml:"root"`
	Runtime    string `yaml:"runtime"`
	RebuildCmd string `yaml:"rebuild_cmd"`
}

type InboxConfig struct {
	RetentionDays int `yaml:"retention_days"`
}

type IndexConfig struct {
	Access AccessConfig `yaml:"access"`
}

type AccessConfig struct {
	HalfLifeDays float64 `yaml:"half_life_days"`
	MinScore     float64 `yaml:"min_score"`
}

type KnowledgeConfig struct {
	Search SearchConfig `yaml:"search"`
}

type SearchConfig struct {
	Weights    map[string]float64            `yaml:"weights"`
	IntentBias map[string]map[string]float64 `yaml:"intent_bias"`
}

type AuditConfig struct {
	RetentionDays int `yaml:"retention_days"`
	RecentLimit   int `yaml:"recent_limit"`
}

type Schema struct {
	VisibilityPolicy         map[string]string   `yaml:"visibility_policy"`
	VisibilityDefault        map[string]string   `yaml:"visibility_default"`
	SensitivePatterns        []string            `yaml:"sensitive_patterns"`
	AuditMinFields           []string            `yaml:"audit_min_fields"`
	AuditResultStatus        []string            `yaml:"audit_result_status"`
	ProposalStates           []string            `yaml:"proposal_states"`
	ProposalStateTransitions map[string][]string `yaml:"proposal_state_transitions"`
	ProposalTargetTypes      []string            `yaml:"proposal_target_types"`
	ProposalOperationTypes   []string            `yaml:"proposal_operation_types"`
	InboxStates              []string            `yaml:"inbox_states"`
	InboxStateTransitions    map[string][]string `yaml:"inbox_state_transitions"`
	IDPrefixes               map[string]string   `yaml:"id_prefixes"`
}

func (c *Config) InboxDir() string     { return filepath.Join(c.Workbase.Runtime, "inbox") }
func (c *Config) ProposalsDir() string { return filepath.Join(c.Workbase.Runtime, "proposals") }
func (c *Config) IndexDB() string {
	return filepath.Join(c.Workbase.Runtime, "index", "notes.sqlite")
}
func (c *Config) AuditDB() string {
	return filepath.Join(c.Workbase.Runtime, "audit", "audit.sqlite")
}
func (c *Config) AuthDB() string { return filepath.Join(c.Workbase.Runtime, "auth.sqlite") }
func (c *Config) TemplatesDir() string {
	return filepath.Join(c.Workbase.Runtime, "templates")
}

func (c *Config) RuntimeDirs() []string {
	return []string{
		c.Workbase.Runtime,
		filepath.Dir(c.IndexDB()),
		c.ProposalsDir(),
		c.InboxDir(),
		filepath.Dir(c.AuditDB()),
		c.TemplatesDir(),
	}
}

// Load 读 yaml。path 空则用 ./config.yaml。文件不存在或必填缺失都返回 error，不静默 Default。
func Load(path string) (*Config, error) {
	if path == "" {
		path = "config.yaml"
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config.yaml: %w", err)
	}
	c := &Config{}
	if err := yaml.Unmarshal(raw, c); err != nil {
		return nil, fmt.Errorf("parse config.yaml: %w", err)
	}
	applyDefaults(c)
	resolvePaths(c, path)
	if err := validate(c); err != nil {
		return nil, err
	}
	return c, nil
}

func applyDefaults(c *Config) {
	if c.Server.Listen == "" {
		c.Server.Listen = DefaultServerListen
	}
	if c.Admin.Listen == "" {
		c.Admin.Listen = DefaultAdminListen
	}
	if c.Admin.SessionTTL == 0 {
		c.Admin.SessionTTL = DefaultSessionTTL
	}
	if c.Admin.LoginRateLimit == 0 {
		c.Admin.LoginRateLimit = DefaultLoginRateLimit
	}
	if c.Workbase.Runtime == "" {
		c.Workbase.Runtime = DefaultRuntime
	}
	if c.Inbox.RetentionDays == 0 {
		c.Inbox.RetentionDays = DefaultInboxRetentionDays
	}
	if c.Index.Access.HalfLifeDays == 0 {
		c.Index.Access.HalfLifeDays = DefaultAccessHalfLifeDays
	}
	if c.Index.Access.MinScore == 0 {
		c.Index.Access.MinScore = DefaultAccessMinScore
	}
	if c.Audit.RetentionDays == 0 {
		c.Audit.RetentionDays = DefaultAuditRetentionDays
	}
	if c.Audit.RecentLimit == 0 {
		c.Audit.RecentLimit = DefaultAuditRecentLimit
	}
	if c.Schema.SensitivePatterns == nil {
		c.Schema.SensitivePatterns = []string{}
	}
	if len(c.Schema.VisibilityDefault) == 0 {
		c.Schema.VisibilityDefault = DefaultVisibilityDefault()
	}
	if len(c.Knowledge.Search.Weights) == 0 {
		c.Knowledge.Search.Weights = DefaultSearchWeights()
	}
	if c.Knowledge.Search.IntentBias == nil {
		c.Knowledge.Search.IntentBias = DefaultIntentBias()
	}
}

func resolvePaths(c *Config, configPath string) {
	base := filepath.Dir(configPath)
	c.Vault.Root = absFrom(base, c.Vault.Root)
	c.Vault.GitDir = absFrom(base, c.Vault.GitDir)
	c.Workbase.Root = absFrom(base, c.Workbase.Root)
	c.Workbase.Runtime = absFrom(base, c.Workbase.Runtime)
	c.Workbase.RebuildCmd = absFrom(base, c.Workbase.RebuildCmd)
}

func absFrom(base, p string) string {
	if p == "" {
		return ""
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	return filepath.Clean(filepath.Join(base, p))
}

func validate(c *Config) error {
	for _, k := range []string{"public", "private", "secret", "draft"} {
		if c.Schema.VisibilityPolicy == nil || c.Schema.VisibilityPolicy[k] == "" {
			return fmt.Errorf("config.yaml: schema.visibility_policy.%s 不能为空", k)
		}
	}
	if c.AdminAuth.User == "" || c.AdminAuth.PassHash == "" {
		return fmt.Errorf("config.yaml: admin_auth.user / pass_hash 必填")
	}
	return nil
}
