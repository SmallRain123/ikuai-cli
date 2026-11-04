// Package cmd 定义 ikuai-cli 的所有 CLI 命令。
package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/cloud"
	"ikuai-cli/internal/config"
	"ikuai-cli/internal/model"
	"ikuai-cli/internal/output"
)

var (
	cfgPath string
	jsonOut bool
	gwid    string
)

var rootCmd = &cobra.Command{
	Use:   "ikuai-cli",
	Short: "爱快(iKuai)路由器监控与数据查询 CLI",
	Long: `ikuai-cli 是一个用于监控和查询爱快路由器数据的命令行工具。

支持两种接入方式：
  * 云端接入 (cloud)：通过爱快开放平台 Open API 3.0（需 open_id 与 RSA 公钥）
  * 本地接入 (local)：通过 SNMP v1/v2c/v3 直连路由器

常用命令：
  ikuai-cli config init                 生成配置文件模板
  ikuai-cli cloud status  --gwid <id>   查询云端系统状态与实时流量
  ikuai-cli cloud online  --gwid <id>   查询在线用户
  ikuai-cli local  status               查询本地 SNMP 系统信息
  ikuai-cli local  traffic              查询本地接口流量速率
  ikuai-cli monitor --source cloud      持续监控模式
`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

func init() {
	rootCmd.PersistentFlags().StringVar(&cfgPath, "config", "", "配置文件路径（默认 ~/.ikuai-cli/config.yaml）")
	rootCmd.PersistentFlags().BoolVar(&jsonOut, "json", false, "以 JSON 格式输出")
	rootCmd.PersistentFlags().StringVar(&gwid, "gwid", "", "路由器 gwid（云端必填）")
}

// Execute 运行根命令。
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "错误:", err)
		os.Exit(1)
	}
}

// loadConfig 加载配置文件。
func loadConfig() (*config.Config, error) {
	p := cfgPath
	if p == "" {
		p = config.DefaultPath()
	}
	return config.Load(p)
}

// cloudClient 创建云端客户端。
func cloudClient() (*cloud.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	key, err := cfg.Cloud.CloudPubKey()
	if err != nil {
		return nil, err
	}
	if cfg.Cloud.OpenID == "" {
		return nil, fmt.Errorf("未配置 cloud.open_id，请先运行 `ikuai-cli config init` 并填写配置")
	}
	return cloud.New(cfg.Cloud.OpenID, key, cfg.Cloud.APIBase, cfg.Cloud.TokenLifetime)
}

// requireGwid 校验 gwid 参数。
func requireGwid() (string, error) {
	if gwid == "" {
		return "", fmt.Errorf("请通过 --gwid 指定路由器 gwid（可用 `ikuai-cli cloud devlist` 查看）")
	}
	return gwid, nil
}

// emit 按 jsonOut 决定输出方式：对象或表格。
func emit(v any, headers []string, rows [][]string) error {
	if jsonOut {
		return output.PrintJSON(v)
	}
	output.Table(headers, rows)
	return nil
}

// speedOf 由累计字节差与间隔计算速率。
func speedOf(prev, cur uint64, intervalSec float64) float64 {
	if cur < prev {
		return 0 // 计数器重置
	}
	return float64(cur-prev) / intervalSec
}

func parseF(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

func parseU(s string) uint64 {
	var u uint64
	fmt.Sscanf(s, "%d", &u)
	return u
}

var _ = model.TokenAccess{} // 保持 model 包引用
