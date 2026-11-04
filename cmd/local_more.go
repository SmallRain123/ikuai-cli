package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/config"
	"ikuai-cli/internal/local"
	"ikuai-cli/internal/snmp"
)

// ---------- SNMP 扩展：ARP / 路由 / 客户端 ----------

var arpCmd = &cobra.Command{
	Use:   "arp",
	Short: "查询本地 ARP 表（IP/MAC 映射）",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		cli, err := snmp.New(cfg.Local)
		if err != nil {
			return err
		}
		defer cli.Close()
		list, err := cli.GetARPTable()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, a := range list {
			rows = append(rows, []string{a.IP, a.MAC, fmt.Sprintf("%d", a.IfIndex), a.Type})
		}
		return emit(list, []string{"IP", "MAC", "接口#", "类型"}, rows)
	},
}

var routesCmd = &cobra.Command{
	Use:   "routes",
	Short: "查询本地路由表",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		cli, err := snmp.New(cfg.Local)
		if err != nil {
			return err
		}
		defer cli.Close()
		list, err := cli.GetRouteTable()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, r := range list {
			rows = append(rows, []string{r.Dest, r.Mask, r.NextHop, fmt.Sprintf("%d", r.IfIndex), fmt.Sprintf("%d", r.Metric), r.Protocol})
		}
		return emit(list, []string{"目的", "掩码", "下一跳", "接口#", "度量", "协议"}, rows)
	},
}

var clientsCmd = &cobra.Command{
	Use:   "clients",
	Short: "列出内网客户端（基于 ARP 表）",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		cli, err := snmp.New(cfg.Local)
		if err != nil {
			return err
		}
		defer cli.Close()
		list, err := cli.GetARPTable()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, a := range list {
			if a.Type == "static" || a.Type == "dynamic" {
				rows = append(rows, []string{a.IP, a.MAC, fmt.Sprintf("%d", a.IfIndex), a.Type})
			}
		}
		return emit(list, []string{"IP", "MAC", "接口#", "类型"}, rows)
	},
}

// ---------- 本地 Web 接口（实验性） ----------

var webCmd = &cobra.Command{
	Use:   "web",
	Short: "本地 Web 管理接口（实验性，非官方）",
	Long:  `web 子命令通过本地 Web 管理接口登录爱快路由并查询。该接口为非官方接口，端点和登录算法随固件版本变化，需在真实设备上校准。`,
}

var (
	webLoginPass string
)

var webLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "测试本地 Web 登录",
	RunE: func(cmd *cobra.Command, args []string) error {
		w, cfg, err := newWebClient()
		if err != nil {
			return err
		}
		if webLoginPass != "" {
			cfg.Local.WebPassword = webLoginPass
			w, err = buildWeb(cfg)
			if err != nil {
				return err
			}
		}
		if err := w.Login(); err != nil {
			return err
		}
		fmt.Println("本地 Web 登录成功")
		return nil
	},
}

var callCmd = &cobra.Command{
	Use:   "call <path>",
	Short: "登录后调用任意本地 Web 接口（输出原始 JSON）",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		w, _, err := newWebClient()
		if err != nil {
			return err
		}
		var body any
		if callBody != "" {
			if err := json.Unmarshal([]byte(callBody), &body); err != nil {
				return fmt.Errorf("--body 不是合法 JSON: %w", err)
			}
		}
		raw, err := w.Call(callMethod, args[0], body)
		if err != nil {
			return err
		}
		fmt.Println(string(raw))
		return nil
	},
}

var (
	callMethod string
	callBody   string
	logType    string
	logEndpt   string
	connEndpt  string
)

var logsCmd = &cobra.Command{
	Use:   "logs",
	Short: "查询系统日志（实验性，端点可校准）",
	RunE: func(cmd *cobra.Command, args []string) error {
		w, cfg, err := newWebClient()
		if err != nil {
			return err
		}
		endpoint := logEndpt
		if endpoint == "" {
			endpoint = cfg.Local.LogEndpoint
		}
		var q any
		if logType != "" {
			q = map[string]string{"type": logType}
		}
		raw, err := w.Call("GET", endpoint, q)
		if err != nil {
			return err
		}
		return outputPretty(raw)
	},
}

var conntrackCmd = &cobra.Command{
	Use:   "conntrack",
	Short: "查询连接数（实验性，端点可校准）",
	RunE: func(cmd *cobra.Command, args []string) error {
		w, cfg, err := newWebClient()
		if err != nil {
			return err
		}
		endpoint := connEndpt
		if endpoint == "" {
			endpoint = cfg.Local.ConnEndpoint
		}
		raw, err := w.Call("GET", endpoint, nil)
		if err != nil {
			return err
		}
		return outputPretty(raw)
	},
}

func newWebClient() (*local.WebClient, *config.Config, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, nil, err
	}
	w, err := buildWeb(cfg)
	if err != nil {
		return nil, nil, err
	}
	return w, cfg, nil
}

func buildWeb(cfg *config.Config) (*local.WebClient, error) {
	if cfg.Local.WebUser == "" || cfg.Local.WebPassword == "" {
		return nil, fmt.Errorf("未配置 local.web_user / local.web_password，请先运行 `ikuai-cli config init` 填写")
	}
	baseURL := fmt.Sprintf("http://%s:%d", cfg.Local.RouterIP, cfg.Local.WebPort)
	return local.NewWebClient(baseURL, cfg.Local.WebUser, cfg.Local.WebPassword, cfg.Local.LoginMode)
}

func outputPretty(raw []byte) error {
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		// 非 JSON，直接打印原始内容
		fmt.Println(string(raw))
		return nil
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func init() {
	localCmd.AddCommand(arpCmd, routesCmd, clientsCmd)
	webCmd.AddCommand(webLoginCmd)
	localCmd.AddCommand(webCmd, callCmd, logsCmd, conntrackCmd)

	webLoginCmd.Flags().StringVar(&webLoginPass, "password", "", "覆盖 web_password 测试登录")
	callCmd.Flags().StringVar(&callMethod, "method", "GET", "HTTP 方法 (GET/POST)")
	callCmd.Flags().StringVar(&callBody, "body", "", "POST JSON body")
	logsCmd.Flags().StringVar(&logType, "type", "", "日志类型 (system/operation/attack)")
	logsCmd.Flags().StringVar(&logEndpt, "endpoint", "", "覆盖日志端点")
	conntrackCmd.Flags().StringVar(&connEndpt, "endpoint", "", "覆盖连接数端点")
}
