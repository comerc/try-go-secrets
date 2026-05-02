#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
set -a; [ -f .env ] && source .env; set +a

PUPPETEER_HELPER="$(dirname "$0")/puppeteer.sh"
trap "$PUPPETEER_HELPER stop" EXIT

PUPPETEER_URL="$("$PUPPETEER_HELPER" start >/dev/null; "$PUPPETEER_HELPER" url)"
export PUPPETEER_URL
echo "▶ Puppeteer: $PUPPETEER_URL"

NUM=${1:-""}
if [ -n "$NUM" ]; then
  echo "▶ Генерация видео для файла #$NUM"
  go run ./cmd/main.go -num "$NUM"
else
  echo "▶ Генерация видео для случайного необработанного файла"
  go run ./cmd/main.go
fi
