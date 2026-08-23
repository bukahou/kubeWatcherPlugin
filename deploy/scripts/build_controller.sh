#!/bin/bash
set -e

# ============================================
# 版本号（每次构建自动推送 latest + VERSION 两个 tag）
# - 留空或注释: 只推送 latest
# - v1.x.x:    同时推送 latest + v1.x.x
# ============================================
VERSION="v0.4.8"

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$SCRIPT_DIR/../.."
source "$SCRIPT_DIR/_common.sh"
build_and_push controller
