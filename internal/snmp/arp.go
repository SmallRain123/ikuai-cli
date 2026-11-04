package snmp

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gosnmp/gosnmp"
)

// IP-MIB 相关 OID。
const (
	oidIPNetToMediaTable = "1.3.6.1.2.1.4.22"
	oidIPNetToMediaIf    = "1.3.6.1.2.1.4.22.1.1" // ifIndex
	oidIPNetToMediaPhys  = "1.3.6.1.2.1.4.22.1.2" // MAC
	oidIPNetToMediaNet   = "1.3.6.1.2.1.4.22.1.3" // IP
	oidIPNetToMediaType  = "1.3.6.1.2.1.4.22.1.4" // type

	oidIPRouteTable = "1.3.6.1.2.1.4.21"
	oidIPRouteDest  = "1.3.6.1.2.1.4.21.1.1" // 目的
	oidIPRouteIf    = "1.3.6.1.2.1.4.21.1.2" // 接口
	oidIPRouteMet   = "1.3.6.1.2.1.4.21.1.3" // 度量
	oidIPRouteNext  = "1.3.6.1.2.1.4.21.1.7" // 下一跳
	oidIPRouteType  = "1.3.6.1.2.1.4.21.1.8" // 类型
	oidIPRouteProto = "1.3.6.1.2.1.4.21.1.9" // 协议
	oidIPRouteMask  = "1.3.6.1.2.1.4.21.1.11" // 掩码
)

// ARPEntry ARP 表条目。
type ARPEntry struct {
	IfIndex int    `json:"if_index"`
	IP      string `json:"ip"`
	MAC     string `json:"mac"`
	Type    string `json:"type"`
}

// RouteEntry 路由表条目。
type RouteEntry struct {
	Dest     string `json:"dest"`
	Mask     string `json:"mask"`
	NextHop  string `json:"next_hop"`
	IfIndex  int    `json:"if_index"`
	Metric   uint32 `json:"metric"`
	Protocol string `json:"protocol"`
}

// GetARPTable 遍历 IP-MIB ipNetToMediaTable 获取 ARP 表。
func (c *SNMPClient) GetARPTable() ([]ARPEntry, error) {
	rows := map[string]*ARPEntry{}
	err := c.gw.Walk(oidIPNetToMediaTable, func(pdu gosnmp.SnmpPDU) error {
		col, key := tableRow(pdu.Name, "22.1.")
		if col == "" || key == "" {
			return nil
		}
		row, ok := rows[key]
		if !ok {
			row = &ARPEntry{}
			rows[key] = row
			// key 形如 "<ifIndex>.<ip bytes>"，解析 ifIndex
			row.IfIndex = firstInt(key)
		}
		switch col {
		case oidIPNetToMediaPhys:
			row.MAC = parseMAC(pdu.Value)
		case oidIPNetToMediaNet:
			row.IP = parseIPRow(key)
		case oidIPNetToMediaType:
			row.Type = arpTypeName(toUint64(pdu.Value))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("ARP 表查询失败: %w", err)
	}
	out := make([]ARPEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sortARP(out)
	return out, nil
}

// GetRouteTable 遍历 IP-MIB ipRouteTable 获取路由表。
func (c *SNMPClient) GetRouteTable() ([]RouteEntry, error) {
	rows := map[string]*RouteEntry{}
	err := c.gw.Walk(oidIPRouteTable, func(pdu gosnmp.SnmpPDU) error {
		col, key := tableRow(pdu.Name, "21.1.")
		if col == "" || key == "" {
			return nil
		}
		row, ok := rows[key]
		if !ok {
			row = &RouteEntry{Dest: parseIPRow(key)}
			rows[key] = row
		}
		switch col {
		case oidIPRouteMask:
			row.Mask = parseIPValue(pdu.Value)
		case oidIPRouteDest:
			row.Dest = fmt.Sprintf("%v", pdu.Value)
		case oidIPRouteIf:
			row.IfIndex = int(toUint64(pdu.Value))
		case oidIPRouteMet:
			row.Metric = uint32(toUint64(pdu.Value))
		case oidIPRouteNext:
			row.NextHop = fmt.Sprintf("%v", pdu.Value)
		case oidIPRouteProto:
			row.Protocol = routeProtoName(toUint64(pdu.Value))
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("路由表查询失败: %w", err)
	}
	out := make([]RouteEntry, 0, len(rows))
	for _, r := range rows {
		out = append(out, *r)
	}
	sortRoutes(out)
	return out, nil
}

// tableRow 从 OID 提取列号与行索引。marker 如 "22.1."。
func tableRow(oid, marker string) (col, key string) {
	idx := strings.Index(oid, marker)
	if idx < 0 {
		return "", ""
	}
	rest := oid[idx+len(marker):]
	parts := strings.Split(rest, ".")
	if len(parts) < 2 {
		return "", ""
	}
	col = marker + parts[0]
	key = strings.Join(parts[1:], ".")
	return col, key
}

// parseIPValue 将 OctetString（IPv4）解析为点分十进制。
func parseIPValue(v any) string {
	if b, ok := v.([]byte); ok && len(b) == 4 {
		return fmt.Sprintf("%d.%d.%d.%d", b[0], b[1], b[2], b[3])
	}
	return fmt.Sprintf("%v", v)
}

// parseMAC 将 OctetString 解析为 aa:bb:cc:dd:ee:ff。
func parseMAC(v any) string {
	b, ok := v.([]byte)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	parts := make([]string, 0, len(b))
	for _, x := range b {
		parts = append(parts, fmt.Sprintf("%02x", x))
	}
	return strings.Join(parts, ":")
}

// parseIPRow 从行索引提取 IPv4 地址（key 末尾 4 个数值）。
func parseIPRow(key string) string {
	parts := strings.Split(key, ".")
	if len(parts) < 4 {
		return key
	}
	tail := parts[len(parts)-4:]
	ip := make([]string, 4)
	for i, p := range tail {
		n, _ := strconv.Atoi(p)
		ip[i] = fmt.Sprintf("%d", n)
	}
	return strings.Join(ip, ".")
}

func firstInt(key string) int {
	n, _ := strconv.Atoi(strings.Split(key, ".")[0])
	return n
}

func arpTypeName(v uint64) string {
	switch v {
	case 1:
		return "other"
	case 2:
		return "invalid"
	case 3:
		return "dynamic"
	case 4:
		return "static"
	default:
		return fmt.Sprintf("%d", v)
	}
}

func routeProtoName(v uint64) string {
	switch v {
	case 1:
		return "other"
	case 2:
		return "local"
	case 3:
		return "netmgmt"
	case 4:
		return "icmp"
	case 5:
		return "egp"
	case 6:
		return "ggp"
	case 8:
		return "rip"
	case 9:
		return "ospf"
	case 10:
		return "bgp"
	case 11:
		return "isis"
	case 12:
		return "eigrp"
	default:
		return fmt.Sprintf("%d", v)
	}
}

func sortARP(list []ARPEntry) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].IP < list[i].IP {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}

func sortRoutes(list []RouteEntry) {
	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Dest < list[i].Dest {
				list[i], list[j] = list[j], list[i]
			}
		}
	}
}
