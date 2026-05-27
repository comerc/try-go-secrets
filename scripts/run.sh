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

PUPPETEER_URL="$("$PUPPETEER_HELPER" start >/dev/null; "$PUPPETEER_HELPER" url)"
export PUPPETEER_URL
echo "▶ Puppeteer: $PUPPETEER_URL"

NUM=${1:-""}
tmp_log="$(mktemp)"
trap 'rm -f "$tmp_log"; "$PUPPETEER_HELPER" stop' EXIT

if [ -n "$NUM" ]; then
  echo "▶ Генерация видео для файла #$NUM"
  status=0
  go run ./cmd/main.go -num "$NUM" 2>&1 | tee "$tmp_log" || status=$?
else
  echo "▶ Генерация видео для случайного необработанного файла"
  status=0
  go run ./cmd/main.go 2>&1 | tee "$tmp_log" || status=$?
fi

if [ "$status" -eq 0 ]; then
  exit 0
fi

if grep -q 'invalid_grant' "$tmp_log"; then
  publish_num="$NUM"
  if [ -z "$publish_num" ]; then
    publish_num="$(sed -nE 's/.*Выбран файл: .* \(#([0-9]+)\).*/\1/p' "$tmp_log" | tail -n 1)"
  fi

  echo
  echo "▶ YouTube refresh token протух или отозван. Запускаю OAuth и обновляю .env..."
  go run ./cmd/main.go -youtube-auth-write

  if [ -z "$publish_num" ]; then
    echo "✗ Не смог определить номер ролика для повторной публикации."
    echo "  Запусти вручную: ./scripts/pub.sh <номер>"
    exit "$status"
  fi

  echo
  echo "▶ Повторная публикация готового видео #$publish_num на YouTube"
  go run ./cmd/main.go -pub "$publish_num"
  exit 0
fi

exit "$status"
