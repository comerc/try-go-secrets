#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env
set -a; [ -f .env ] && source .env; set +a

NUM=${1:-""}

if [ -z "$NUM" ]; then
  echo "Использование: ./scripts/public.sh <номер>"
  echo "Пример: ./scripts/public.sh 43"
  exit 1
fi

echo "▶ Публикация готового видео #$NUM на YouTube"

go run ./cmd/main.go -public "$NUM"
