// Package snmp 实现爱快路由器的本地 SNMP 监控接入（v1/v2c/v3）。
package snmp

import (
	"fmt"
	"strings"
	"time"

	"github.com/gosnmp/gosnmp"

	"ikuai-cli/internal/config"
)

// OID 常量（IF-MIB / SYSTEM-MIB）。
const (
	oidSysDescr    = "1.3.6.1.2.1.1.1.0"
	oidSysObjectID = "1.3.6.1.2.1.1.2.0"
	oidSysUpTime   = "1.3.6.1.2.1.1.3.0"
	oidSysContact  = "1.3.6.1.2.1.1.4.0"
	oidSysName     = "1.3.6.1.2.1.1.5.0"
	oidSysLocation = "1.3.6.1.2.1.1.6.0"

	oidIfIndex      = "1.3.6.1.2.1.2.2.1.1"
	oidIfName       = "1.3.6.1.2.1.31.1.1.1.1"
	oidIfOperStatus = "1.3.6.1.2.1.2.2.1.8"
	oidIfInOctets   = "1.3.6.1.2.1.2.2.1.10"
	oidIfOutOctets  = "1.3.6.1.2.1.2.2.1.16"
	oidIfHCIn       = "1.3.6.1.2.1.31.1.1.1.6"
	oidIfHCOut      = "1.3.6.1.2.1.31.1.1.1.10"
)

// SystemInfo 本地 SNMP 系统信息。
type SystemInfo struct {
	Descr    string `json:"descr"`
	ObjectID string `json:"object_id"`
	UpTime   string `json:"up_time"` // 运行时长（秒）
	Contact  string `json:"contact"`
	Name     string `json:"name"`
	Location string `json:"location"`
}

// InterfaceStat 接口流量统计。
type InterfaceStat struct {
	Index      int     `json:"index"`
	Name       string  `json:"name"`
	OperStatus string  `json:"oper_status"`
	InOctets   uint64  `json:"in_octets"`   // 累计入字节
	OutOctets  uint64  `json:"out_octets"`  // 累计出字节
	InSpeed    float64 `json:"in_speed"`    // 入速率 byte/s
	OutSpeed   float64 `json:"out_speed"`   // 出速率 byte/s
}

// SNMPClient 封装 gosnmp 连接。
type SNMPClient struct {
	gw *gosnmp.GoSNMP
}

// New 根据配置创建并建立 SNMP 连接。
func New(lc config.LocalConfig) (*SNMPClient, error) {
	ver, err := parseVersion(lc.SNMPVersion)
	if err != nil {
		return nil, err
	}
	gw := &gosnmp.GoSNMP{
		Target:    lc.RouterIP,
		Port:      lc.Port,
		Version:   ver,
		Timeout:   time.Duration(6) * time.Second,
		Retries:   2,
		MaxOids:   60,
	}
	switch ver {
	case gosnmp.Version1, gosnmp.Version2c:
		gw.Community = lc.Community
	case gosnmp.Version3:
		gw.SecurityModel = gosnmp.UserSecurityModel
		gw.MsgFlags = gosnmp.AuthPriv
		gw.SecurityParameters = &gosnmp.UsmSecurityParameters{
			UserName:                 lc.Username,
			AuthenticationProtocol:   gosnmp.SHA,
			AuthenticationPassphrase: lc.AuthPassword,
			PrivacyProtocol:          gosnmp.AES,
			PrivacyPassphrase:        lc.PrivPassword,
		}
	}
	if err := gw.Connect(); err != nil {
		return nil, fmt.Errorf("SNMP 连接 %s:%d 失败: %w", lc.RouterIP, lc.Port, err)
	}
	return &SNMPClient{gw: gw}, nil
}

func parseVersion(v string) (gosnmp.SnmpVersion, error) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "", "v2c", "2c", "2":
		return gosnmp.Version2c, nil
	case "v1", "1":
		return gosnmp.Version1, nil
	case "v3", "3":
		return gosnmp.Version3, nil
	default:
		return 0, fmt.Errorf("不支持的 SNMP 版本: %s（支持 v1/v2c/v3）", v)
	}
}

// Close 关闭连接。
func (c *SNMPClient) Close() { _ = c.gw.Conn.Close() }

// GetSystemInfo 查询系统信息。
func (c *SNMPClient) GetSystemInfo() (*SystemInfo, error) {
	pkt, err := c.gw.Get([]string{
		oidSysDescr, oidSysObjectID, oidSysUpTime,
		oidSysContact, oidSysName, oidSysLocation,
	})
	if err != nil {
		return nil, fmt.Errorf("SNMP 系统信息查询失败: %w", err)
	}
	info := &SystemInfo{}
	for _, v := range pkt.Variables {
		val := asString(v)
		switch v.Name {
		case oidSysDescr:
			info.Descr = val
		case oidSysObjectID:
			info.ObjectID = val
		case oidSysUpTime:
			info.UpTime = val
		case oidSysContact:
			info.Contact = val
		case oidSysName:
			info.Name = val
		case oidSysLocation:
			info.Location = val
		}
	}
	return info, nil
}

// GetInterfaces 遍历 IF-MIB，返回各接口累计流量。
func (c *SNMPClient) GetInterfaces() ([]InterfaceStat, error) {
	ifaces := map[int]*InterfaceStat{}
	// 建立 index -> 结构
	if err := c.walk(oidIfIndex, func(oid string, val any) {
		idx := indexOf(oid)
		ifaces[idx] = &InterfaceStat{Index: idx}
	}); err != nil {
		return nil, err
	}

	apply := func(baseOID string, set func(*InterfaceStat, uint64)) error {
		return c.walk(baseOID, func(oid string, val any) {
			idx := indexOf(oid)
			if s, ok := ifaces[idx]; ok {
				set(s, toUint64(val))
			}
		})
	}
	// ifName 是 OctetString，单独处理
	if err := c.walkString(oidIfName, func(oid string, val string) {
		if s, ok := ifaces[indexOf(oid)]; ok {
			s.Name = val
		}
	}); err != nil {
		return nil, err
	}
	if err := apply(oidIfOperStatus, func(s *InterfaceStat, v uint64) { s.OperStatus = statusName(v) }); err != nil {
		return nil, err
	}
	if err := apply(oidIfHCIn, func(s *InterfaceStat, v uint64) { s.InOctets = v }); err != nil {
		return nil, err
	}
	if err := apply(oidIfHCOut, func(s *InterfaceStat, v uint64) { s.OutOctets = v }); err != nil {
		return nil, err
	}

	out := make([]InterfaceStat, 0, len(ifaces))
	for _, s := range ifaces {
		out = append(out, *s)
	}
	sortIfaces(out)
	return out, nil
}

func (c *SNMPClient) walk(oid string, fn func(oid string, val any)) error {
	return c.gw.Walk(oid, func(pdu gosnmp.SnmpPDU) error {
		fn(pdu.Name, pdu.Value)
		return nil
	})
}

func (c *SNMPClient) walkString(oid string, fn func(oid, val string)) error {
	return c.gw.Walk(oid, func(pdu gosnmp.SnmpPDU) error {
		fn(pdu.Name, asString(pdu))
		return nil
	})
}

func indexOf(oid string) int {
	parts := strings.Split(oid, ".")
	if len(parts) == 0 {
		return 0
	}
	var n int
	for _, p := range parts[len(parts)-1:] {
		fmt.Sscanf(p, "%d", &n)
	}
	return n
}

func asString(pdu gosnmp.SnmpPDU) string {
	switch pdu.Type {
	case gosnmp.OctetString:
		return string(pdu.Value.([]byte))
	default:
		return fmt.Sprintf("%v", pdu.Value)
	}
}

func toUint64(v any) uint64 {
	switch t := v.(type) {
	case uint:
		return uint64(t)
	case uint64:
		return t
	case int:
		return uint64(t)
	case int64:
		return uint64(t)
	case string:
		var out uint64
		fmt.Sscanf(t, "%d", &out)
		return out
	}
	return 0
}

func statusName(v uint64) string {
	switch v {
	case 1:
		return "up"
	case 2:
		return "down"
	case 3:
		return "testing"
	default:
		return fmt.Sprintf("unknown(%d)", v)
	}
}

func sortIfaces(list []InterfaceStat) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Index < list[i].Index {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}
