#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

while true; do
  echo "━━━ $(date '+%H:%M:%S') Запуск пайплайна ━━━"
  "$SCRIPT_DIR/run.sh" || echo "⚠  run.sh завершился с ошибкой, перезапуск..."
  echo "━━━ $(date '+%H:%M:%S') Пауза 2с ━━━"
  sleep 2
done
