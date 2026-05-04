#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
VIDEO_LANG_OVERRIDE="${VIDEO_LANG:-}"
YOUTUBE_ENABLED_OVERRIDE="${YOUTUBE_ENABLED:-}"
set -a; [ -f .env ] && source .env; set +a
if [ -n "$VIDEO_LANG_OVERRIDE" ]; then
  export VIDEO_LANG="$VIDEO_LANG_OVERRIDE"
fi
if [ -n "$YOUTUBE_ENABLED_OVERRIDE" ]; then
  export YOUTUBE_ENABLED="$YOUTUBE_ENABLED_OVERRIDE"
fi

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
