#!/usr/bin/env bash
# ikuai-cli 简易流量监控循环（依赖系统 curl，无需 Go）
# 用法: ./scripts/monitor-snmp.sh <路由IP> <community> [间隔秒]
# 通过 SNMP 查询接口出/入字节计数，并计算实时速率。
set -euo pipefail

IP="${1:?用法: $0 <路由IP> <community> [间隔秒]}"
COMMUNITY="${2:-public}"
INTERVAL="${3:-5}"
OID_IF="1.3.6.1.2.1.31.1.1.1"
OID_IN="$OID_IF.6"   # ifHCOutOctets
OID_OUT="$OID_IF.10" # ifHCInOctets

need() { command -v "$1" >/dev/null 2>&1 || { echo "缺少命令: $1" >&2; exit 1; }; }
need snmpget

prev_in=""
prev_out=""
prev_ts=""

fetch() { # $1=OID -> 数值
  snmpget -v2c -c "$COMMUNITY" -On "$IP" "$1" 2>/dev/null \
    | awk -F': ' 'END{gsub(/Gauge32:|Counter64:|Counter32:/,"",$2); gsub(/ /,"",$2); print $2}'
}

hr() { # 字节 -> 人类可读
  local n="$1"
  if (( n < 1024 )); then echo "${n} B"; 
  elif (( n < 1048576 )); then awk "BEGIN{printf \"%.2f KiB\", $n/1024}"
  elif (( n < 1073741824 )); then awk "BEGIN{printf \"%.2f MiB\", $n/1048576}"
  else awk "BEGIN{printf \"%.2f GiB\", $n/1073741824}"; fi
}

speed() { # 每秒字节 -> 人类可读速率
  awk "BEGIN{ r=$1*8; if(r<1000) printf \"%.0f bps\", r; else if(r<1e6) printf \"%.2f Kbps\", r/1e3; else printf \"%.2f Mbps\", r/1e6 }"
}

echo "开始监控 $IP (community=$COMMUNITY, 间隔=${INTERVAL}s)  Ctrl-C 退出"
for (( ;; )); do
  in=$(fetch "$OID_IN")
  out=$(fetch "$OID_OUT")
  ts=$(date +%s)

  if [[ -n "$prev_in" && -n "$prev_out" ]]; then
    dt=$((ts - prev_ts))
    if (( dt > 0 )); then
      in_bps=$(( (in - prev_in) / dt ))
      out_bps=$(( (out - prev_out) / dt ))
      printf "[%s] 下行(入): %-12s  上行(出): %-12s  累计入: %-12s  累计出: %s\n" \
        "$(date '+%H:%M:%S')" "$(speed "$in_bps")" "$(speed "$out_bps")" "$(hr "$in")" "$(hr "$out")"
    fi
  else
    echo "首轮采样完成，开始计算速率..."
  fi

  prev_in=$in; prev_out=$out; prev_ts=$ts
  sleep "$INTERVAL"
done
