#!/usr/bin/env bash
# ikuai-cli 一键构建与初始化脚本
# 用法: ./scripts/ikuai-cli-setup.sh [安装目录]
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
INSTALL_DIR="${1:-$HOME/bin}"
BIN_NAME="ikuai-cli"

echo "[1/3] 检查 Go 环境..."
if ! command -v go >/dev/null 2>&1; then
  echo "错误: 未安装 Go (https://go.dev/dl)" >&2
  exit 1
fi

echo "[2/3] 编译 ${BIN_NAME}..."
(cd "$REPO_DIR" && go build -trimpath -ldflags "-s -w -X ikuai-cli/cmd.version=$(git describe --tags --always 2>/dev/null || echo dev)" -o "$BIN_NAME" .)

mkdir -p "$INSTALL_DIR"
mv -f "$REPO_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
echo "    已安装到: $INSTALL_DIR/$BIN_NAME"

echo "[3/3] 初始化配置文件..."
"$INSTALL_DIR/$BIN_NAME" config init 2>/dev/null || echo "    配置文件已存在，跳过生成"

echo
echo "完成！配置写入 ~/.ikuai-cli/config.yaml 后即可使用："
echo "  $INSTALL_DIR/$BIN_NAME cloud status --gwid <id>"
echo "  $INSTALL_DIR/$BIN_NAME local status"
