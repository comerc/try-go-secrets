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

NUM=${1:-""}

if [ -z "$NUM" ]; then
  echo "Использование: ./scripts/pub.sh <номер>"
  echo "Пример: ./scripts/pub.sh 43"
  exit 1
fi

echo "▶ Публикация готового видео #$NUM на YouTube"

go run ./cmd/main.go -pub "$NUM"
