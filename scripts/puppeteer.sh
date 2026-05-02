#!/usr/bin/env bash
set -euo pipefail

PUPPETEER_DIR="$(cd "$(dirname "$0")/.." && pwd)/puppeteer"
PUPPETEER_PID_FILE="${PUPPETEER_PID_FILE:-/tmp/go-secrets-puppeteer.pid}"
PUPPETEER_LOG_FILE="${PUPPETEER_LOG_FILE:-/tmp/go-secrets-puppeteer.log}"
PUPPETEER_INSTALL_LOG_FILE="${PUPPETEER_INSTALL_LOG_FILE:-/tmp/go-secrets-puppeteer-install.log}"
PUPPETEER_URL_FILE="${PUPPETEER_URL_FILE:-/tmp/go-secrets-puppeteer.url}"

port_is_free() {
  ! lsof -nP -iTCP:"$1" -sTCP:LISTEN >/dev/null 2>&1
}

pick_free_port() {
  if [ -n "${PUPPETEER_PORT:-}" ] && port_is_free "$PUPPETEER_PORT"; then
    echo "$PUPPETEER_PORT"
    return 0
  fi

  if [ -n "${PUPPETEER_PORT:-}" ]; then
    echo "⚠ Порт $PUPPETEER_PORT занят, ищу свободный..." >&2
  fi

  for _ in $(seq 1 100); do
    port=$((RANDOM % 45535 + 10000))
    if port_is_free "$port"; then
      echo "$port"
      return 0
    fi
  done

  echo "Не удалось найти свободный порт для Puppeteer" >&2
  exit 1
}

is_healthy() {
  curl -fsS "$PUPPETEER_URL/health" >/dev/null 2>&1
}

start_puppeteer() {
  if [ -f "$PUPPETEER_URL_FILE" ]; then
    PUPPETEER_URL="$(cat "$PUPPETEER_URL_FILE")"
    PUPPETEER_PORT="${PUPPETEER_URL##*:}"
    PUPPETEER_PORT="${PUPPETEER_PORT%%/*}"
  else
    PUPPETEER_PORT="$(pick_free_port)"
    PUPPETEER_URL="http://127.0.0.1:$PUPPETEER_PORT"
    printf '%s\n' "$PUPPETEER_URL" > "$PUPPETEER_URL_FILE"
  fi

  if is_healthy; then
    printf '%s\n' "$PUPPETEER_URL"
    exit 0
  fi

  if [ -f "$PUPPETEER_URL_FILE" ]; then
    rm -f "$PUPPETEER_URL_FILE"
    PUPPETEER_PORT="$(pick_free_port)"
    PUPPETEER_URL="http://127.0.0.1:$PUPPETEER_PORT"
    printf '%s\n' "$PUPPETEER_URL" > "$PUPPETEER_URL_FILE"
  fi

  echo "▶ Запуск Puppeteer на хосте ($PUPPETEER_URL)" >&2
  if [ ! -d "$PUPPETEER_DIR/node_modules" ]; then
    (cd "$PUPPETEER_DIR" && npm install >/dev/null 2>"$PUPPETEER_INSTALL_LOG_FILE")
  fi

  (cd "$PUPPETEER_DIR" && PORT="$PUPPETEER_PORT" npm start >/dev/null 2>"$PUPPETEER_LOG_FILE" & echo $! > "$PUPPETEER_PID_FILE")

  for _ in $(seq 1 30); do
    if is_healthy; then
      printf '%s\n' "$PUPPETEER_URL"
      exit 0
    fi
    sleep 1
  done

  echo "✗ Puppeteer не поднялся, см. $PUPPETEER_LOG_FILE" >&2
  exit 1
}

print_url() {
  if [ -f "$PUPPETEER_URL_FILE" ]; then
    cat "$PUPPETEER_URL_FILE"
    return 0
  fi

  echo "Puppeteer URL неизвестен" >&2
  exit 1
}

stop_puppeteer() {
  if [ -f "$PUPPETEER_PID_FILE" ]; then
    pid="$(cat "$PUPPETEER_PID_FILE")"
    kill "$pid" >/dev/null 2>&1 || true
    rm -f "$PUPPETEER_PID_FILE"
  fi
  rm -f "$PUPPETEER_URL_FILE"
}

status_puppeteer() {
  if is_healthy; then
    echo "ok"
  else
    echo "down"
    exit 1
  fi
}

case "${1:-}" in
  start) start_puppeteer ;;
  stop) stop_puppeteer ;;
  status) status_puppeteer ;;
  url) print_url ;;
  *)
    echo "Использование: $0 {start|stop|status|url}"
    exit 1
    ;;
esac
