package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"ikuai-cli/internal/output"
)

var cloudCmd = &cobra.Command{
	Use:   "cloud",
	Short: "通过爱快开放平台 Open API 3.0 查询（需 open_id + RSA 公钥）",
	Long:  "cloud 子命令通过爱快开放平台 Open API 3.0 查询路由器数据。需要先配置 cloud.open_id 与 cloud.rsa_pub_key(_path)。",
}

var tokenCmd = &cobra.Command{
	Use:   "token",
	Short: "获取并输出 AccessToken",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cloudClient()
		if err != nil {
			return err
		}
		if err := c.GetToken(); err != nil {
			return err
		}
		return output.PrintJSON(map[string]any{
			"access_token": c.Token(),
			"expires_at":   c.TokenExpireAt(),
		})
	},
}

var devlistCmd = &cobra.Command{
	Use:   "devlist",
	Short: "查看授权设备列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		c, err := cloudClient()
		if err != nil {
			return err
		}
		list, err := c.DeviceList()
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, d := range list {
			rows = append(rows, []string{d.DevID, d.DevRemark, d.OwnerMobile, fmt.Sprintf("%v", d.OpenAPIs)})
		}
		return emit(list, []string{"gwid", "备注", "所属手机", "开放接口"}, rows)
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查询路由器系统状态与实时流量",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireGwid()
		if err != nil {
			return err
		}
		c, err := cloudClient()
		if err != nil {
			return err
		}
		st, err := c.SystemStatus(id)
		if err != nil {
			return err
		}
		if jsonOut {
			return output.PrintJSON(st)
		}
		output.KV("设备名", st.Name)
		output.KV("gwid", st.Gwid)
		output.KV("MAC", st.Mac)
		output.KV("固件", st.Firmware)
		output.KV("路由版本", st.RouterVer)
		output.KV("运行时长", st.SysUpTime+" s")
		output.KV("在线用户", st.Online)
		output.KV("AP 数量", st.APCount)
		output.KV("上行速率", output.HumanSpeed(parseF(st.UpSpeed)*1024))
		output.KV("下行速率", output.HumanSpeed(parseF(st.DownSpeed)*1024))
		output.KV("累计上行", output.HumanBytes(parseU(st.TotalUp)))
		output.KV("累计下行", output.HumanBytes(parseU(st.TotalDown)))
		output.KV("公网出口", st.ReflexiveAddr)
		return nil
	},
}

var onlineCmd = &cobra.Command{
	Use:   "online",
	Short: "查询在线用户列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireGwid()
		if err != nil {
			return err
		}
		c, err := cloudClient()
		if err != nil {
			return err
		}
		list, err := c.OnlineUsers(id, 0, 0)
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, u := range list {
			rows = append(rows, []string{
				u.IPAddr, u.Mac, u.Username, u.Name,
				output.HumanSpeed(parseF(u.Down)), output.HumanSpeed(parseF(u.Up)),
				u.Interface, u.ConnectTime,
			})
		}
		return emit(list, []string{"IP", "MAC", "用户名", "实名", "下行", "上行", "接口", "在线时长"}, rows)
	},
}

var accountsCmd = &cobra.Command{
	Use:   "accounts",
	Short: "查询认证账号列表",
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := requireGwid()
		if err != nil {
			return err
		}
		c, err := cloudClient()
		if err != nil {
			return err
		}
		list, err := c.Accounts(id)
		if err != nil {
			return err
		}
		rows := make([][]string, 0, len(list))
		for _, a := range list {
			rows = append(rows, []string{a.Username, a.Name, a.IPAddr, a.Mac, a.Enabled, a.PppType, a.Comment})
		}
		return emit(list, []string{"账号", "姓名", "IP", "MAC", "启用", "类型", "备注"}, rows)
	},
}

func init() {
	cloudCmd.AddCommand(tokenCmd, devlistCmd, statusCmd, onlineCmd, accountsCmd)
	rootCmd.AddCommand(cloudCmd)
}
