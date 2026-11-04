// Package config 负责 ikuai-cli 配置的加载与默认值。
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// CloudConfig 云端开放平台接入配置。
type CloudConfig struct {
	OpenID         string `yaml:"open_id"`           // 开放平台 open_id（即 client_id）
	RSAPubKey      string `yaml:"rsa_pub_key"`       // RSA 公钥 PEM 内容（或使用公钥文件路径）
	RSAPubKeyPath  string `yaml:"rsa_pub_key_path"`  // RSA 公钥 PEM 文件路径
	TokenLifetime  int64  `yaml:"token_lifetime"`    // AccessToken 有效期（秒），默认 7200
	APIBase        string `yaml:"api_base"`          // API 基础路径，默认 https://open.ikuai8.com/api/v3
}

// LocalConfig 本地接入配置（SNMP / 本地 Web 接口）。
type LocalConfig struct {
	RouterIP     string `yaml:"router_ip"`     // 路由管理 IP，默认 192.168.9.1
	Port         uint16 `yaml:"port"`          // 默认 161
	SNMPVersion  string `yaml:"snmp_version"`  // v1 / v2c / v3，默认 v2c
	Community    string `yaml:"community"`     // v1/v2c community，默认 public
	Username     string `yaml:"username"`      // v3 用户名
	AuthProtocol string `yaml:"auth_protocol"` // v3 认证协议 MD5/SHA，默认 SHA
	AuthPassword string `yaml:"auth_password"` // v3 认证密码
	PrivProtocol string `yaml:"priv_protocol"` // v3 加密协议 AES/DES，默认 AES
	PrivPassword string `yaml:"priv_password"` // v3 加密密码
	WebUser      string `yaml:"web_user"`      // 本地 Web 管理账号（实验性）
	WebPassword  string `yaml:"web_password"`  // 本地 Web 管理密码（实验性）
	WebPort      uint16 `yaml:"web_port"`      // 本地 Web 端口，默认 80
	LoginMode    string `yaml:"login_mode"`    // 本地 Web 登录密码模式：md5 / plain / auto
	LogEndpoint  string `yaml:"log_endpoint"`  // 日志查询端点，默认 /api/log/list
	ConnEndpoint string `yaml:"conn_endpoint"` // 连接数查询端点，默认 /api/conntrack/table
}

// MonitorConfig 监控模式参数。
type MonitorConfig struct {
	Interval int `yaml:"interval"` // 刷新间隔（秒），默认 5
	Count    int `yaml:"count"`    // 刷新次数，0=持续
}

// Config 顶层配置。
type Config struct {
	Cloud   CloudConfig   `yaml:"cloud"`
	Local   LocalConfig   `yaml:"local"`
	Monitor MonitorConfig `yaml:"monitor"`
}

// Default 返回带默认值的配置。
func Default() *Config {
	return &Config{
		Cloud: CloudConfig{
			TokenLifetime: 7200,
			APIBase:       "https://open.ikuai8.com/api/v3",
		},
		Local: LocalConfig{
			RouterIP:     "192.168.9.1",
			Port:         161,
			SNMPVersion:  "v2c",
			Community:    "public",
			AuthProtocol: "SHA",
			PrivProtocol: "AES",
		},
		Monitor: MonitorConfig{Interval: 5, Count: 0},
	}
}

// Load 从 path 加载配置；path 为空时返回默认配置。
// 若文件不存在且给了 path，则视为错误（应使用 init 子命令生成）。
func Load(path string) (*Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
	}
	cfg.fillDefaults()
	return cfg, nil
}

func (c *Config) fillDefaults() {
	if c.Cloud.TokenLifetime <= 0 {
		c.Cloud.TokenLifetime = 7200
	}
	if c.Cloud.APIBase == "" {
		c.Cloud.APIBase = "https://open.ikuai8.com/api/v3"
	}
	if c.Local.Port == 0 {
		c.Local.Port = 161
	}
	if c.Local.SNMPVersion == "" {
		c.Local.SNMPVersion = "v2c"
	}
	if c.Local.Community == "" {
		c.Local.Community = "public"
	}
	if c.Local.WebPort == 0 {
		c.Local.WebPort = 80
	}
	if c.Local.LoginMode == "" {
		c.Local.LoginMode = "auto"
	}
	if c.Local.LogEndpoint == "" {
		c.Local.LogEndpoint = "/api/log/list"
	}
	if c.Local.ConnEndpoint == "" {
		c.Local.ConnEndpoint = "/api/conntrack/table"
	}
	if c.Monitor.Interval <= 0 {
		c.Monitor.Interval = 5
	}
}

// CloudPubKey 解析并返回 RSA 公钥 PEM 内容。
func (c *CloudConfig) CloudPubKey() (string, error) {
	if c.RSAPubKey != "" {
		return c.RSAPubKey, nil
	}
	if c.RSAPubKeyPath != "" {
		data, err := os.ReadFile(c.RSAPubKeyPath)
		if err != nil {
			return "", fmt.Errorf("读取 RSA 公钥文件失败: %w", err)
		}
		return string(data), nil
	}
	return "", fmt.Errorf("未配置 cloud.rsa_pub_key 或 cloud.rsa_pub_key_path")
}

// DefaultPath 返回默认配置文件绝对路径。
func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "ikuai-cli.yaml"
	}
	return filepath.Join(home, ".ikuai-cli", "config.yaml")
}
