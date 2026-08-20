// Package config 读取 config.yaml（含 schema 块）。
package config

// 可调参数缺省。cfg 有值用 cfg，没值用这里。一份数字只写一次。
const (
	DefaultServerListen = "127.0.0.1:8787"
	DefaultAdminListen  = "127.0.0.1:8788"

	DefaultSessionTTL     = 3600
	DefaultLoginRateLimit = 5

	DefaultInboxRetentionDays = 7

	DefaultAccessHalfLifeDays = 7
	DefaultAccessMinScore     = 0.001

	DefaultAuditRetentionDays = 90
	DefaultAuditRecentLimit   = 100

	DefaultRuntime = ".workbase"

	IdentityID      = "jiangnan-workbase"
	IdentityVersion = "0.1.0"
)

// DefaultSearchWeights 对应 SCHEMA §1 knowledge.search.weights。
func DefaultSearchWeights() map[string]float64 {
	return map[string]float64{
		"title":            5.0,
		"tags":             4.0,
		"frontmatter":      3.0,
		"section":          2.0,
		"fulltext":         1.5,
		"wikilink_backref": 2.0,
		"access":           1.0,
		"recency":          0.5,
	}
}

// DefaultIntentBias 对应 SCHEMA §1 knowledge.search.intent_bias。
func DefaultIntentBias() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"why":     {"frontmatter": 1.3, "section": 1.3},
		"when":    {"recency": 1.5},
		"entity":  {"tags": 1.3},
		"general": {},
	}
}

// DefaultVisibilityDefault 对应 SCHEMA §3.2。运行时权威仍是 yaml；这里只给缺省 map。
func DefaultVisibilityDefault() map[string]string {
	return map[string]string{
		"文章":               "public",
		"项目":               "public",
		"友链":               "public",
		"部署溯源":             "private",
		"Workbase/context": "private",
		"Workbase/skills":  "private",
		"Workbase/mcps":    "private",
		"default":          "private",
	}
}
