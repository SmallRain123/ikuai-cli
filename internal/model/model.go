// Package model 定义 ikuai-cli 统一的数据模型。
package model

// SystemStatus 路由器基本信息与实时流量（云端 Open API 3.0 /open/9/{gwid}）。
type SystemStatus struct {
	Name           string `json:"name"`            // 设备名
	Gwid           string `json:"gwid"`            // 设备唯一标识
	Mac            string `json:"mac"`             // MAC
	RouterVer      string `json:"router_ver"`      // 路由版本
	Firmware       string `json:"firmware"`        // 固件
	LibVer         string `json:"libver"`          // 库版本
	SysUpTime      string `json:"sysuptime"`       // 运行时长（秒）
	APCount        string `json:"ap_count"`        // AP 数量
	Online         string `json:"online"`          // 在线用户数
	UpSpeed        string `json:"up_speed"`        // 上行速率 (KB/s)
	DownSpeed      string `json:"down_speed"`      // 下行速率 (KB/s)
	TotalUp        string `json:"total_up"`        // 累计上行 (byte)
	TotalDown      string `json:"total_down"`      // 累计下行 (byte)
	ReflexiveAddr  string `json:"reflexive_address"` // 公网出口 IP
	Enterprise     string `json:"enterprise"`
	Partner        string `json:"partner"`
}

// OnlineUser 在线用户（Open API 3.0 /open/4/{gwid}）。
type OnlineUser struct {
	ID         string `json:"id"`
	IP         string `json:"ip"`          // 会话 IP
	IPAddr     string `json:"ip_addr"`     // 认证 IP
	Mac        string `json:"mac"`
	Type       string `json:"type"`        // 终端类型 0=未知/1=PC/...
	Username   string `json:"username"`
	Name       string `json:"name"`        // 实名
	Comment    string `json:"comment"`
	Up         string `json:"up"`          // 上行速率 (byte/s)
	Down       string `json:"down"`        // 下行速率 (byte/s)
	TotalUp    string `json:"total_up"`    // 累计上行
	TotalDown  string `json:"total_down"`  // 累计下行
	Upload     string `json:"upload"`
	Download   string `json:"download"`
	Interface  string `json:"interface"`   // 接口
	ConnectTime string `json:"connect_time"` // 在线时长
	Connect    string `json:"connect"`     // 连接数
	Expires    string `json:"expires"`
	Phone      string `json:"phone"`
	PppType    string `json:"ppptype"`
	PppDev     string `json:"pppdev"`
	Session    string `json:"session"`
	AuthTime   string `json:"auth_time"`
	WebID      string `json:"webid"`
}

// Account 认证账号（Open API 3.0 /open/1/{gwid}）。
type Account struct {
	ID         string `json:"id"`
	Username   string `json:"username"`
	Name       string `json:"name"`
	Passwd     string `json:"passwd"`
	Comment    string `json:"comment"`
	IPAddr     string `json:"ip_addr"`
	Mac        string `json:"mac"`
	Phone      string `json:"phone"`
	Enabled    string `json:"enabled"`   // yes/no
	PppType    string `json:"ppptype"`   // pppoe/pptp/l2tp...
	BindIfname string `json:"bind_ifname"`
	BindVlanID string `json:"bind_vlanid"`
	AutoMac    string `json:"auto_mac"`
	AutoVlanID string `json:"auto_vlanid"`
	Upload     string `json:"upload"`    // 限速 上行
	Download   string `json:"download"`  // 限速 下行
	Share      string `json:"share"`
	Expires    string `json:"expires"`
	StartTime  string `json:"start_time"`
	CreateTime string `json:"create_time"`
	LastConnTime string `json:"last_conntime"`
	CardID     string `json:"cardid"`
	Address    string `json:"address"`
	Packages   string `json:"packages"`
}

// Device 授权设备（Open API 3.0 /open/dev_list）。
type Device struct {
	DevID      string   `json:"dev_id"`
	DevRemark  string   `json:"dev_remark"`
	OwnerMobile string  `json:"owner_mobile"`
	OwnerQQ    string   `json:"owner_qq"`
	OpenAPIs   []string `json:"open_apis"`
}

// TokenAccess 访问令牌（云端认证）。
type TokenAccess struct {
	AccessToken        string `json:"access_token"`
	TokenCreateTime    int64  `json:"token_create_time"`
	TokenExcessTime    int64  `json:"token_excess_time"`
	TokenExpireAtServer string `json:"token_expire_at_server_time"`
}
