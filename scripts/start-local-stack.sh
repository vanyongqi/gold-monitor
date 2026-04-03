#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="$ROOT_DIR/tmp"
PID_DIR="$ROOT_DIR/tmp"
DB_PATH="$ROOT_DIR/data/gold_monitor.db"
HTTP_PORT="${HTTP_PORT:-8090}"
COLLECT_INTERVAL="${COLLECT_INTERVAL:-60s}"
BIN_PATH="$ROOT_DIR/bin/gold-monitor"

mkdir -p "$LOG_DIR" "$PID_DIR" "$ROOT_DIR/data" "$ROOT_DIR/bin"

start_if_needed() {
  local name="$1"
  local pid_file="$2"
  shift 2

  if [[ -f "$pid_file" ]]; then
    local existing_pid
    existing_pid="$(cat "$pid_file")"
    if kill -0 "$existing_pid" 2>/dev/null; then
      echo "$name 已在运行，PID=$existing_pid"
      return
    fi
    rm -f "$pid_file"
  fi

  nohup "$@" >"$LOG_DIR/${name}.log" 2>&1 &
  local new_pid=$!
  echo "$new_pid" >"$pid_file"
  echo "$name 已启动，PID=$new_pid"
}

echo "构建本地二进制..."
go build -o "$BIN_PATH" "$ROOT_DIR/cmd/gold-monitor"

start_if_needed \
  "collector" \
  "$PID_DIR/collector.pid" \
  "$BIN_PATH" -collector -db "$DB_PATH" -interval "$COLLECT_INTERVAL"

start_if_needed \
  "web" \
  "$PID_DIR/web.pid" \
  "$BIN_PATH" -http -listen ":$HTTP_PORT" -db "$DB_PATH"

echo "网页地址: http://localhost:$HTTP_PORT"
echo "日志目录: $LOG_DIR"
