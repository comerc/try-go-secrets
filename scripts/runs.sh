#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

# Загружаем .env, но явно переданный VIDEO_LANGS имеет приоритет.
VIDEO_LANGS_OVERRIDE="${VIDEO_LANGS:-}"
YOUTUBE_ENABLED_OVERRIDE="${YOUTUBE_ENABLED:-}"
set -a; [ -f .env ] && source .env; set +a
if [ -n "$VIDEO_LANGS_OVERRIDE" ]; then
  export VIDEO_LANGS="$VIDEO_LANGS_OVERRIDE"
fi
if [ -n "$YOUTUBE_ENABLED_OVERRIDE" ]; then
  export YOUTUBE_ENABLED="$YOUTUBE_ENABLED_OVERRIDE"
fi

if [ -z "${VIDEO_LANGS:-}" ]; then
  echo "VIDEO_LANGS не задан"
  echo "Пример: VIDEO_LANGS=ru,en-us,de ./scripts/runs.sh"
  exit 1
fi

IFS=',' read -ra LANGS <<< "$VIDEO_LANGS"
failed=0

for lang in "${LANGS[@]}"; do
  lang="$(printf '%s' "$lang" | xargs)"
  if [ -z "$lang" ]; then
    continue
  fi

  echo
  echo "━━━ VIDEO_LANG=$lang ━━━"
  if VIDEO_LANG="$lang" ./scripts/run.sh "$@"; then
    echo "✓ VIDEO_LANG=$lang готово"
  else
    status=$?
    failed=$((failed + 1))
    echo "✗ VIDEO_LANG=$lang завершился с ошибкой (exit $status), продолжаю..."
  fi
done

if [ "$failed" -gt 0 ]; then
  echo
  echo "✗ Завершено с ошибками: $failed"
  exit 1
fi
