package cmd

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/output"
	"ikuai-cli/internal/snmp"
)

var localCmd = &cobra.Command{
	Use:   "local",
	Short: "通过 SNMP 直连本地路由器查询（v1/v2c/v3）",
	Long:  "local 子命令通过 SNMP 直连路由器查询，需先在爱快后台开启 SNMP 服务并配置 local 相关参数。",
}

var localStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询本地 SNMP 系统信息",
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
		info, err := cli.GetSystemInfo()
		if err != nil {
			return err
		}
		if jsonOut {
			return output.PrintJSON(info)
		}
		output.KV("系统名", info.Name)
		output.KV("设备类型", info.Descr)
		output.KV("运行时长", info.UpTime+" s")
		output.KV("位置", info.Location)
		output.KV("联系人", info.Contact)
		output.KV("ObjectID", info.ObjectID)
		return nil
	},
}

var localInterfacesCmd = &cobra.Command{
	Use:   "interfaces",
	Short: "查询本地接口状态与累计流量",
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
		list, err := cli.GetInterfaces()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, it := range list {
			rows = append(rows, []string{
				fmt.Sprintf("%d", it.Index), it.Name, it.OperStatus,
				output.HumanBytes(it.InOctets), output.HumanBytes(it.OutOctets),
			})
		}
		return emit(list, []string{"#", "接口", "状态", "累计入", "累计出"}, rows)
	},
}

var (
	trafficInterval int
)

var localTrafficCmd = &cobra.Command{
	Use:   "traffic",
	Short: "查询接口实时速率（两次采样计算）",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		if trafficInterval <= 0 {
			trafficInterval = 1
		}
		cli, err := snmp.New(cfg.Local)
		if err != nil {
			return err
		}
		defer cli.Close()

		first, err := cli.GetInterfaces()
		if err != nil {
			return err
		}
		time.Sleep(time.Duration(trafficInterval) * time.Second)
		second, err := cli.GetInterfaces()
		if err != nil {
			return err
		}

		sec := float64(trafficInterval)
		byIdx := map[int]snmp.InterfaceStat{}
		for _, it := range second {
			byIdx[it.Index] = it
		}
		rows := make([][]string, 0, len(first))
		out := make([]snmp.InterfaceStat, 0, len(first))
		for _, it := range first {
			cur, ok := byIdx[it.Index]
			if !ok {
				continue
			}
			cur.InSpeed = speedOf(it.InOctets, cur.InOctets, sec)
			cur.OutSpeed = speedOf(it.OutOctets, cur.OutOctets, sec)
			out = append(out, cur)
			rows = append(rows, []string{
				fmt.Sprintf("%d", cur.Index), cur.Name, cur.OperStatus,
				output.HumanSpeed(cur.InSpeed), output.HumanSpeed(cur.OutSpeed),
			})
		}
		return emit(out, []string{"#", "接口", "状态", "入速率", "出速率"}, rows)
	},
}

func init() {
	localTrafficCmd.Flags().IntVar(&trafficInterval, "interval", 1, "采样间隔（秒）")
	localCmd.AddCommand(localStatusCmd, localInterfacesCmd, localTrafficCmd)
	rootCmd.AddCommand(localCmd)
}
