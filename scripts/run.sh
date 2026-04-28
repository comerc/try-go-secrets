#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
set -a; [ -f .env ] && source .env; set +a

VIA_DOCKER="${VIA_DOCKER:-false}"
NUM=${1:-""}

# Puppeteer всегда через docker
docker compose up -d --build puppeteer

if [ "$VIA_DOCKER" = "true" ]; then
  # ── Pipeline в Docker ───────────────────────────────────────────
  if [ -n "$NUM" ]; then
    echo "▶ Генерация видео для файла #$NUM"
    docker compose run --rm --build pipeline -num "$NUM"
  else
    echo "▶ Генерация видео для случайного необработанного файла"
    docker compose run --rm --build pipeline
  fi

else
  # ── Pipeline локально ───────────────────────────────────────────
  # PUPPETEER_URL из .env уже указывает на localhost:3333 — используем как есть
  if [ -n "$NUM" ]; then
    echo "▶ Генерация видео для файла #$NUM (локально, puppeteer=docker)"
    go run ./cmd/main.go -num "$NUM"
  else
    echo "▶ Генерация видео для случайного необработанного файла (локально, puppeteer=docker)"
    go run ./cmd/main.go
  fi
fi
