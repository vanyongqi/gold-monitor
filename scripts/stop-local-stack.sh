#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

stop_pid_file() {
  local name="$1"
  local pid_file="$2"
  if [[ ! -f "$pid_file" ]]; then
    echo "$name 未运行"
    return
  fi

  local pid
  pid="$(cat "$pid_file")"
  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid"
    echo "$name 已停止，PID=$pid"
  else
    echo "$name 进程不存在，清理 PID 文件"
  fi
  rm -f "$pid_file"
}

stop_pid_file "web" "$ROOT_DIR/tmp/web.pid"
stop_pid_file "collector" "$ROOT_DIR/tmp/collector.pid"
