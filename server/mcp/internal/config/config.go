// Package config 读取 server/mcp 的 config.yaml。
package config

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Vault    VaultConfig    `yaml:"vault"`
	Workbase WorkbaseConfig `yaml:"workbase"`
	Admin    AdminConfig    `yaml:"admin"`
	Auth     AuthConfig     `yaml:"auth"`
}

type ServerConfig struct {
	Listen string `yaml:"listen"`
}

type VaultConfig struct {
	Root   string `yaml:"root"`
	GitDir string `yaml:"git_dir"`
}

type WorkbaseConfig struct {
	Root       string `yaml:"root"`
	Index      string `yaml:"index"`
	Proposals  string `yaml:"proposals"`
	Inbox      string `yaml:"inbox"`
	Audit      string `yaml:"audit"`
	RebuildCmd string `yaml:"rebuild_cmd"` // apply 后可选：博客 rebuild 命令（空则跳过）
}

type AdminConfig struct {
	Listen string    `yaml:"listen"`
	Auth   AdminAuth `yaml:"auth"`
}

type AdminAuth struct {
	User     string `yaml:"user"`
	PassHash string `yaml:"pass_hash"`
}

type AuthConfig struct {
	Clients []Client `yaml:"clients"`
}

type Client struct {
	ID        string   `yaml:"id"`
	TokenHash string   `yaml:"token_hash"`
	Scopes    []string `yaml:"scopes"`
}

// Default 返回一套开发期默认值，避免 config.yaml 缺失时直接 panic。
func Default() *Config {
	return &Config{
		Server: ServerConfig{Listen: "127.0.0.1:8787"},
		Vault:  VaultConfig{Root: "D:/Data/工作台", GitDir: ""},
		Workbase: WorkbaseConfig{
			Root:      "D:/Data/工作台/Workbase",
			Index:     "./.workbase/index",
			Proposals: "./.workbase/proposals",
			Inbox:     "./.workbase/inbox",
			Audit:     "./.workbase/audit.jsonl",
		},
		Admin: AdminConfig{Listen: "127.0.0.1:8788"},
	}
}

// Load 读取指定路径的 YAML 配置；文件不存在时返回 Default()。
func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return Default(), nil
		}
		return nil, err
	}
	c := Default()
	if err := yaml.Unmarshal(b, c); err != nil {
		return nil, err
	}
	return c, nil
}
