#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
set -a; [ -f .env ] && source .env; set +a

VIA_DOCKER="${VIA_DOCKER:-false}"
NUM=${1:-""}

if [ -z "$NUM" ]; then
  echo "Использование: ./scripts/fix.sh <номер>"
  echo "Пример: ./scripts/fix.sh 181"
  exit 1
fi

echo "▶ Перегенерация аудио+видео для #$NUM (без z.ai)"

# Puppeteer всегда через docker
docker compose up -d --build puppeteer

if [ "$VIA_DOCKER" = "true" ]; then
  # ── Pipeline в Docker ───────────────────────────────────────────
  docker compose run --rm --build pipeline -fix "$NUM"

else
  # ── Pipeline локально ───────────────────────────────────────────
  go run ./cmd/main.go -fix "$NUM"
fi
