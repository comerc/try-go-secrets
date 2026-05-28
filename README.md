# Go Secrets Pipeline

Автоматизированный пайплайн генерации YouTube Shorts (<60 сек) из 286 markdown-файлов с секретами Go.

**Стек:** Go · Gemini TTS · Codex CLI / Claude Code CLI / z.ai · Puppeteer · FFmpeg

---

## Режимы запуска

В `.env` используются ключевые параметры запуска:

| Переменная | Значение | Эффект |
|---|---|---|
| `LLM_BACKEND` | `codex-cli` (по умолчанию) | инференс через локальный Codex CLI |
| `LLM_BACKEND` | `claude-cli` | инференс через локальный Claude Code CLI |
| `LLM_BACKEND` | `zai-api` | инференс через z.ai API |
| `LLM_MODEL` | например `gpt-5.5` | модель для Codex, Claude и z.ai |
| `LLM_EFFORT` | `medium` по умолчанию | уровень reasoning/thinking для LLM-бэкенда |
| `PUPPETEER_URL` | выбирается автоматически, если не задан | адрес локально запущенного Puppeteer-сервиса |

Поддерживаемые уровни `LLM_EFFORT` зависят от бэкенда:

- `claude-cli`: `low`, `medium`, `high`, `xhigh`, `max`
- `codex-cli`: `low`, `medium`, `high`, `xhigh`
- `zai-api`: `low`, `medium`, `high`

Для `zai-api` значение `LLM_EFFORT` маппится так:

- `low` -> `thinking.disabled`
- `medium` -> `thinking.enabled`
- `high` -> `thinking.enabled` + `clear_thinking=false`

---

## Быстрый старт (локальный режим)

```bash
# 1. Скопировать и заполнить ключи API
cp .env.example .env

# 2. Убедиться что Codex CLI залогинен
codex login

# 3. Конкретный файл по номеру
./scripts/run.sh 43
# → output/ru/videos/YYYY-MM-DD__NNN.mp4

# Другой язык на один запуск
VIDEO_LANG=es ./scripts/run.sh 43

# 4. Случайный необработанный файл
./scripts/run.sh

# 5. Бесконечный цикл (все файлы подряд)
./scripts/loop.sh
```

`run.sh` запускает pipeline локально (`go run ./cmd/main.go`). Если Puppeteer ещё не установлен, скрипт сам сделает `npm install` в `puppeteer/`, поднимет сервис на свободном порту и дождётся `/health`.

Результаты:
- `output/<VIDEO_LANG>/` — видео, аудио, сценарии
- `state/tts_usage.json` — общий учёт TTS
- `state/<VIDEO_LANG>/` — `processed.json`, `youtube_schedule.json`

---

## YouTube автопубликация

Финальный шаг пайплайна может автоматически загрузить готовый ролик в YouTube, поставить публикацию в расписание и добавить видео в плейлист.

Включается через `.env`:

```bash
YOUTUBE_ENABLED=true
YOUTUBE_CLIENT_ID=...
YOUTUBE_CLIENT_SECRET=...
YOUTUBE_REFRESH_TOKEN=...
YOUTUBE_PLAYLIST_ID_RU=...
TERMINAL_TITLE_RU=300 секретов Golang
YOUTUBE_SCHEDULE_LOCATION_RU=Europe/Moscow
YOUTUBE_SCHEDULE_TIME_RU=12:00
```

Поведение:
- `Script.Title` отправляется как title ролика.
- `Script.NarrationText` отправляется в description.
- `TERMINAL_TITLE_<VIDEO_LANG>` отображается в заголовке терминала внутри видео.
- Видео загружается как `private` с `publishAt`, то есть YouTube сам опубликует его в заданное время.
- Публикация планируется по одному ролику в день. Занятые даты хранятся в `state/<VIDEO_LANG>/youtube_schedule.json`.
- Ролик добавляется в плейлист `YOUTUBE_PLAYLIST_ID_<VIDEO_LANG>`.

Опубликовать уже готовый ролик без перегенерации:

```bash
./scripts/pub.sh 43
```

Скрипт найдёт последние `output/<VIDEO_LANG>/scripts/*__043.json` и `output/<VIDEO_LANG>/videos/*__043.mp4`, выберет первую свободную дату и загрузит ролик в YouTube.

`state/<VIDEO_LANG>/youtube_schedule.json` хранит только номер ролика и дату:

```json
{
  "043": "2026-05-03"
}
```

Для OAuth refresh token нужен доступ к YouTube Data API v3 со scope `https://www.googleapis.com/auth/youtube`.

---

## Исправление готового ролика (fix)

Если нужно поправить озвучку/субтитры без перегенерации сценария через LLM:

**1. Найти сценарий**

```
output/ru/scripts/YYYY-MM-DD__NNN.json
```

**2. Отредактировать поля**

| Поле | Что менять |
|---|---|
| `NarrationTags` | Gemini TTS markup tags для озвучки, например `[short pause]`, `[whispering]`; официальный список: https://docs.cloud.google.com/text-to-speech/docs/gemini-tts#markup_tag_guide |
| `NarrationText` | Чистый текст субтитров (без Audio-Tags) |
| `Segments[].Text` | Текст отдельного сегмента субтитров |

**3. Запустить перегенерацию**

```bash
./scripts/fix.sh 43
```

Скрипт пропускает LLM, берёт отредактированный `NarrationTags` и заново синтезирует аудио + рендерит видео (Puppeteer + FFmpeg).

---

## Разработка

```bash
# Unit-тесты (парсер + выбор контента), без внешних зависимостей
go test ./tests/...

# Тест TTS изолированно
go run ./cmd/main.go -test-tts "Привет мир"
# → output/ru/audio/test-tts.wav

# Puppeteer поднимается автоматически через `./scripts/run.sh` и `./scripts/fix.sh`
```

---

## Переменные окружения

| Переменная | Описание |
|---|---|
| `LLM_BACKEND` | `codex-cli` — Codex CLI, `claude-cli` — Claude Code CLI, `zai-api` — z.ai API |
| `LLM_MODEL` | модель для Codex, Claude и z.ai (`glm-5.1` по умолчанию для z.ai) |
| `LLM_EFFORT` | уровень effort/thinking для LLM-бэкенда |
| `ZAI_API_KEY` | ключ z.ai API |
| `GEMINI_API_KEY` | ключ Gemini API для TTS |
| `GEMINI_TTS_MODEL` | модель Gemini TTS, по умолчанию `gemini-2.5-flash-preview-tts` |
| `GEMINI_TTS_VOICE` | голос Gemini TTS; если пусто, выбирается случайно из `x-voices.md` |
| `PUPPETEER_URL` | URL локального puppeteer-сервиса, подбирается автоматически |
| `RAW_DIR` | папка с markdown-файлами, по умолчанию `./raw` |
| `OUTPUT_DIR` | базовая папка с результатами, по умолчанию `./output`; язык добавляется автоматически |
| `STATE_DIR` | базовая папка с состоянием, по умолчанию `./state`; язык добавляется автоматически |
| `VIDEO_LANG` | язык нарратива и подпапок: `ru` (по умолчанию), `en-us`, `es` |
| `VIDEO_WIDTH` | ширина видео в пикселях, по умолчанию `1080` |
| `VIDEO_HEIGHT` | высота видео в пикселях, по умолчанию `1920` |
| `VIDEO_FPS` | частота кадров, по умолчанию `30` |
| `TERMINAL_TITLE_<LANG>` | текст в заголовке терминала внутри видео, например `TERMINAL_TITLE_EN_US` |
| `YOUTUBE_ENABLED` | включает финальную загрузку и планирование YouTube |
| `YOUTUBE_CLIENT_ID` | OAuth client id Google Cloud |
| `YOUTUBE_CLIENT_SECRET` | OAuth client secret Google Cloud |
| `YOUTUBE_REFRESH_TOKEN` | refresh token для YouTube Data API |
| `YOUTUBE_PLAYLIST_ID_<LANG>` | ID плейлиста для языка, например `YOUTUBE_PLAYLIST_ID_RU` |
| `YOUTUBE_SCHEDULE_LOCATION_<LANG>` | IANA-таймзона расписания, например `YOUTUBE_SCHEDULE_LOCATION_RU=Europe/Moscow` |
| `YOUTUBE_SCHEDULE_TIME_<LANG>` | время публикации по выбранной таймзоне в формате `HH:MM`, например `YOUTUBE_SCHEDULE_TIME_RU=13:30` |

---

YOUTUBE_ENABLED=true ./scripts/runs.sh
