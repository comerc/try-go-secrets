#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")/.."

LIMIT="${TTS_DAILY_LIMIT:-10}"
USAGE_FILE="state/tts_usage.json"

today() {
  date +%F
}

current_runs() {
  local day="$1"

  if [ ! -f "$USAGE_FILE" ]; then
    echo 0
    return
  fi

  local entry
  entry="$(sed -nE 's/^[[:space:]]*"'"$day"'"[[:space:]]*:[[:space:]]*"([0-9]+)-[0-9]+".*/\1/p' "$USAGE_FILE" | tail -n 1)"
  if [ -z "$entry" ]; then
    echo 0
    return
  fi

  echo "$entry"
}

day="$(today)"
runs="$(current_runs "$day")"

echo "▶ TTS usage за $day: $runs/$LIMIT"

while [ "$runs" -lt "$LIMIT" ]; do
  echo
  echo "━━━ VIDEO_LANG=ru, запуск $((runs + 1))/$LIMIT ━━━"

  if VIDEO_LANG=ru ./scripts/run.sh "$@"; then
    runs="$(current_runs "$day")"
    echo "✓ TTS usage за $day: $runs/$LIMIT"
  else
    status=$?
    echo "✗ ./scripts/run.sh завершился с ошибкой (exit $status)"
    exit "$status"
  fi

  if [ "$runs" -ge "$LIMIT" ]; then
    break
  fi
done

echo
echo "✓ Лимит TTS на $day достигнут: $runs/$LIMIT"
