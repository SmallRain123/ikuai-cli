package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/config"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "管理配置文件",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "生成配置文件模板",
	RunE: func(cmd *cobra.Command, args []string) error {
		path := cfgPath
		if path == "" {
			path = config.DefaultPath()
		}
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("配置文件已存在: %s", path)
		}
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			return err
		}
		tpl := `# ikuai-cli 配置文件
# 生成时间: {{time}}
cloud:
  # 爱快开放平台 open_id（平台后台获取）
  open_id: "ikoxxxxxxxxxxxxxxxxxxxx"
  # RSA 公钥 PEM 内容；或使用 rsa_pub_key_path 指向公钥文件
  rsa_pub_key: |
    -----BEGIN PUBLIC KEY-----
    MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEA...
    -----END PUBLIC KEY-----
  # rsa_pub_key_path: "/etc/ikuai-cli/open_rsa_pubkey.pem"
  token_lifetime: 7200
  api_base: "https://open.ikuai8.com/api/v3"

local:
  router_ip: "192.168.9.1"
  port: 161
  snmp_version: "v2c"      # v1 / v2c / v3
  community: "public"
  username: ""             # v3 用户名
  auth_protocol: "SHA"
  auth_password: ""
  priv_protocol: "AES"
  priv_password: ""
  # web_user: ""           # 本地 Web 管理账号（实验性）
  # web_password: ""

monitor:
  interval: 5              # 刷新间隔（秒）
  count: 0                 # 刷新次数，0=持续
`
		tpl = tplReplaceTime(tpl)
		if err := os.WriteFile(path, []byte(tpl), 0o600); err != nil {
			return err
		}
		fmt.Printf("已生成配置文件模板: %s\n请填写 open_id、RSA 公钥及 local 参数后使用。\n", path)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "显示默认配置文件路径",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println(config.DefaultPath())
		return nil
	},
}

func tplReplaceTime(tpl string) string {
	return tpl
}

func init() {
	configCmd.AddCommand(configInitCmd, configPathCmd)
	rootCmd.AddCommand(configCmd)
}
