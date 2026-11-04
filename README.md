# ikuai-cli

爱快（iKuai）路由器监控与数据查询命令行工具。支持 **云端开放平台 Open API 3.0**、**本地 SNMP v1/v2c/v3** 与 **本地 Web 管理接口（实验性）** 三种接入方式，用于查询系统状态、实时流量、在线用户、认证账号、接口速率、ARP 表、内网客户端、路由表与日志。

纯 Go 实现，编译为单一二进制，无运行时依赖；附 Shell 辅助监控脚本。

## 特性

- **三种接入通道**
  - 云端：爱快开放平台 Open API 3.0（RSA 签名获取/刷新 AccessToken）
  - 本地 SNMP：v1/v2c/v3 直连路由器，无需开放平台账号
  - 本地 Web（实验性）：登录爱快 Web 管理界面查询日志、连接数等
- **监控能力**：系统状态、接口实时速率、在线用户、认证账号、授权设备、ARP 表、内网客户端、路由表、日志、连接数
- **持续监控模式** `monitor`：按周期实时刷新状态/流量
- **输出灵活**：对齐表格或 JSON（`--json`）
- **可校准**：本地 Web 端点可用 `local call` 通用调用自适配
- **可审计**：RSA 签名逻辑带单元测试；接口调用有完整错误提示

## 快速开始

前置依赖：Go 1.24+（仅构建期需要）。

```bash
# 一键构建并安装到 ~/bin
./scripts/ikuai-cli-setup.sh

# 或手动构建
go build -o ikuai-cli .

# 生成配置文件模板
ikuai-cli config init
```

生成的配置位于 `~/.ikuai-cli/config.yaml`，按需填写云端或本地参数后即可使用。

## 配置

```yaml
cloud:                          # 云端接入（二选一填公钥内容或文件路径）
  open_id: "ikoxxxxxxxxxxxx"    # 开放平台 open_id
  rsa_pub_key: |               # RSA 公钥 PEM 内容
    -----BEGIN PUBLIC KEY-----
    ...
    -----END PUBLIC KEY-----
  rsa_pub_key_path: ""          # 或指向公钥文件
  token_lifetime: 7200
  api_base: "https://open.ikuai8.com/api/v3"

local:                          # 本地接入（SNMP + 可选 Web 接口）
  router_ip: "192.168.9.1"
  port: 161
  snmp_version: "v2c"           # v1 / v2c / v3
  community: "public"
  username: ""                  # v3 用户名
  auth_protocol: "SHA"
  auth_password: ""
  priv_protocol: "AES"
  priv_password: ""
  web_user: ""                  # 本地 Web 管理账号（实验性）
  web_password: ""              # 本地 Web 管理密码
  web_port: 80                  # 本地 Web 端口（https 通常为 443）
  web_scheme: "http"            # 本地 Web 协议 http/https
  web_insecure: true            # 忽略 HTTPS 证书校验（爱快自签证书建议 true）
  login_mode: "auto"            # Web 登录密码模式：md5 / plain / auto
  log_endpoint: "/api/log/list"           # 日志查询端点（可校准）
  conn_endpoint: "/api/conntrack/table"   # 连接数查询端点（可校准）

monitor:
  interval: 5                   # 刷新间隔（秒）
  count: 0                      # 刷新次数，0=持续
```

## 命令参考

| 命令 | 说明 |
|---|---|
| `config init` | 生成配置文件模板 |
| `config path` | 显示默认配置路径 |
| `cloud token` | 获取并输出 AccessToken |
| `cloud devlist` | 查看授权设备列表 |
| `cloud status --gwid <id>` | 查询系统状态与实时流量 |
| `cloud online --gwid <id>` | 查询在线用户 |
| `cloud accounts --gwid <id>` | 查询认证账号 |
| `local status` | 查询本地 SNMP 系统信息 |
| `local interfaces` | 查询本地接口状态与累计流量 |
| `local traffic [--interval N]` | 查询接口实时速率（两次采样） |
| `local arp` | 查询本地 ARP 表（IP/MAC 映射） |
| `local clients` | 列出内网客户端（基于 ARP 表） |
| `local routes` | 查询本地路由表 |
| `local web login [--password]` | 测试本地 Web 登录（实验性） |
| `local call <path> [--method --body]` | 登录后调用任意本地 Web 接口 |
| `local logs [--type system\|operation\|attack]` | 查询日志（实验性，端点可校准） |
| `local conntrack` | 查询连接数（实验性，端点可校准） |
| `monitor --source cloud\|local` | 持续监控模式 |
| `version` | 打印版本 |

所有查询命令支持 `--json` 以 JSON 输出。

## 使用示例

```bash
# 云端：查看授权设备，获取 gwid
ikuai-cli cloud devlist

# 云端：查询某设备状态与实时流量
ikuai-cli cloud status --gwid <id>
ikuai-cli cloud status --gwid <id> --json

# 云端：在线用户 / 账号
ikuai-cli cloud online --gwid <id>
ikuai-cli cloud accounts --gwid <id>

# 本地：系统信息 / 接口流量 / ARP / 客户端 / 路由
ikuai-cli local status
ikuai-cli local traffic --interval 2
ikuai-cli local arp
ikuai-cli local clients
ikuai-cli local routes

# 本地 Web（实验性）：登录测试、任意调用、日志、连接数
# 爱快 Web 管理默认使用 HTTPS 自签证书，内网访问请加 --insecure（或 config web_insecure: true）
ikuai-cli local web login --scheme https --insecure
ikuai-cli local call /api/log/list --scheme https --insecure
ikuai-cli local logs --type operation --scheme https --insecure
ikuai-cli local conntrack --scheme https --insecure

# 持续监控
ikuai-cli monitor --source local
ikuai-cli monitor --source cloud --gwid <id> --count 10
```

> 本地 Web 接口为非官方接口，端点可能随固件版本变化。若默认端点不对，可用 `local call <path>` 探测实际接口，再用 `local logs --endpoint <path>` / `local conntrack --endpoint <path>` 校准。

## 云端接入准备

1. 登录 [爱快开放平台](https://open.ikuai8.com/)，申请合作方账号并绑定路由设备。
2. 获取 `open_id` 与 `open_rsa_pubkey`（RSA 公钥）。
3. 将公钥写入配置（`rsa_pub_key` 或 `rsa_pub_key_path`）。
4. 通过 `ikuai-cli cloud devlist` 获取 `gwid`，即可查询。

## 本地 SNMP 配置（爱快后台）

1. 登录路由器 Web 管理界面。
2. 进入 **系统设置 → 服务应用 → SNMP**（或 **设备管理 → SNMP**）。
3. 开启 SNMP 服务，选择版本并设置 community（v2c）或认证/加密参数（v3）。
4. 将参数填入 `local` 配置，即可使用 `local` 系列命令。

## 测试与构建

```bash
make vet test build   # 静态检查 + 单元测试 + 编译
```

单元测试覆盖 RSA 公钥解析（PKCS1/PKIX）、签名正确性（私钥反向验签）、输出单位换算等。

## 目录结构

```
ikuai-cli/
├── main.go              # 入口
├── cmd/                 # cobra 子命令（cloud/local/monitor/config/version）
├── internal/
│   ├── cloud/           # 云端 Open API 3.0（RSA 认证 + 业务接口）
│   ├── local/           # 本地 Web 管理接口接入（登录 + 通用调用，实验性）
│   ├── snmp/            # SNMP 系统信息、接口流量、ARP、路由
│   ├── config/          # yaml 配置加载
│   ├── output/          # 表格 / JSON / 单位换算
│   └── model/           # 数据模型
├── scripts/             # 一键安装脚本、SNMP 监控脚本
├── docs/                # 文档
└── Makefile
```

## 备注

- 本地 SNMP 提供查询类监控能力（系统/接口/ARP/路由/客户端），为只读操作。
- 本地 Web 接口（日志/连接数）为非官方接口，实验性，端点和登录算法可能随固件版本变化，需在真实设备上校准；登录密码模式可用 `login_mode` 调整，接口端点用 `local call` 探测后用 `--endpoint` 校准。
- 云端接口遵循爱快开放平台频率限制（见官方文档）。
