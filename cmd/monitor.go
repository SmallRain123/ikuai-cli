package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/config"
	"ikuai-cli/internal/output"
	"ikuai-cli/internal/snmp"
)

var (
	monitorSource string
	monitorCount  int
)

var monitorCmd = &cobra.Command{
	Use:   "monitor",
	Short: "持续监控模式（实时刷新系统状态/流量）",
	Long: `持续监控模式，按配置的 interval 周期刷新并打印状态。

示例：
  ikuai-cli monitor --source cloud --gwid <id>   # 云端监控
  ikuai-cli monitor --source local                # 本地 SNMP 监控
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		interval := cfg.Monitor.Interval
		if interval <= 0 {
			interval = 5
		}
		count := monitorCount
		if count == 0 {
			count = -1 // 持续
		}

		switch monitorSource {
		case "cloud":
			return monitorCloud(cfg, interval, count)
		case "local":
			return monitorLocal(cfg, interval, count)
		default:
			return fmt.Errorf("未知监控源 %q（支持 cloud / local）", monitorSource)
		}
	},
}

func monitorCloud(cfg *config.Config, interval, count int) error {
	id, err := requireGwid()
	if err != nil {
		return err
	}
	c, err := cloudClient()
	if err != nil {
		return err
	}
	n := 0
	for {
		st, err := c.SystemStatus(id)
		if err != nil {
			fmt.Fprintln(os.Stderr, "查询失败:", err)
			return err
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("[%s] 在线=%s 上行=%s 下行=%s 累计上=%s 累计下=%s\n",
			ts, st.Online,
			output.HumanSpeed(parseF(st.UpSpeed)*1024),
			output.HumanSpeed(parseF(st.DownSpeed)*1024),
			output.HumanBytes(parseU(st.TotalUp)),
			output.HumanBytes(parseU(st.TotalDown)))
		n++
		if count > 0 && n >= count {
			return nil
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func monitorLocal(cfg *config.Config, interval, count int) error {
	cli, err := snmp.New(cfg.Local)
	if err != nil {
		return err
	}
	defer cli.Close()
	first, err := cli.GetInterfaces()
	if err != nil {
		return err
	}
	n := 0
	for {
		second, err := cli.GetInterfaces()
		if err != nil {
			return err
		}
		sec := float64(interval)
		byIdx := map[int]snmp.InterfaceStat{}
		for _, it := range second {
			byIdx[it.Index] = it
		}
		ts := time.Now().Format("2006-01-02 15:04:05")
		fmt.Printf("\n[%s] 接口速率 (间隔 %ds):\n", ts, interval)
		for _, it := range first {
			cur, ok := byIdx[it.Index]
			if !ok {
				continue
			}
			in := speedOf(it.InOctets, cur.InOctets, sec)
			out := speedOf(it.OutOctets, cur.OutOctets, sec)
			if it.Name != "" {
				fmt.Printf("  %-16s 入=%-12s 出=%s\n", it.Name, output.HumanSpeed(in), output.HumanSpeed(out))
			}
		}
		first = second
		n++
		if count > 0 && n >= count {
			return nil
		}
		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func init() {
	monitorCmd.Flags().StringVar(&monitorSource, "source", "cloud", "监控源：cloud / local")
	monitorCmd.Flags().IntVar(&monitorCount, "count", 0, "刷新次数（0=持续）")
	rootCmd.AddCommand(monitorCmd)
}
