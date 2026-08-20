// Package search 放 knowledge.search 的默认权重。yaml 有值用 yaml。
package search

const (
	DefaultLimit = 10
	MaxLimit     = 50
)

func DefaultWeights() map[string]float64 {
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

func DefaultIntentBias() map[string]map[string]float64 {
	return map[string]map[string]float64{
		"why":     {"frontmatter": 1.3, "section": 1.3},
		"when":    {"recency": 1.5},
		"entity":  {"tags": 1.3},
		"general": {},
	}
}

func MergeWeights(over map[string]float64) map[string]float64 {
	out := DefaultWeights()
	for k, v := range over {
		if v != 0 {
			out[k] = v
		}
	}
	return out
}

func IntentBias(all map[string]map[string]float64, intent string) map[string]float64 {
	if all == nil {
		all = DefaultIntentBias()
	}
	src := all[intent]
	out := map[string]float64{}
	for k, v := range src {
		out[k] = v
	}
	return out
}
