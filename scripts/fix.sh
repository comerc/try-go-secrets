#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
set -a; [ -f .env ] && source .env; set +a

NUM=${1:-""}
PUPPETEER_HELPER="$(dirname "$0")/puppeteer.sh"
trap "$PUPPETEER_HELPER stop" EXIT

if [ -z "$NUM" ]; then
  echo "Использование: ./scripts/fix.sh <номер>"
  echo "Пример: ./scripts/fix.sh 181"
  exit 1
fi

echo "▶ Перегенерация аудио+видео для #$NUM (без LLM)"

PUPPETEER_URL="$("$PUPPETEER_HELPER" start >/dev/null; "$PUPPETEER_HELPER" url)"
export PUPPETEER_URL
echo "▶ Puppeteer: $PUPPETEER_URL"

go run ./cmd/main.go -fix "$NUM"
