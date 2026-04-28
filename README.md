# Go Secrets Pipeline

Автоматизированный пайплайн генерации YouTube Shorts (<60 сек) из 286 markdown-файлов с секретами Go.

**Стек:** Go · SaluteSpeech TTS · z.ai / Claude Code CLI · Puppeteer · FFmpeg · Docker

---

## Режимы запуска

В `.env` два флага, которые управляют режимом:

| Переменная | Значение | Эффект |
|---|---|---|
| `VIA_DOCKER` | `false` (по умолчанию) | pipeline запускается локально через `go run` |
| `VIA_DOCKER` | `true` | pipeline запускается в Docker-контейнере |
| `LLM_BACKEND` | `claude-cli` (по умолчанию) | инференс через локальный Claude Code CLI |
| `LLM_BACKEND` | `zai` | инференс через z.ai API (GLM) |

> **Puppeteer** всегда запускается через `docker compose` — независимо от `VIA_DOCKER`.

---

## Быстрый старт (локальный режим)

```bash
# 1. Скопировать и заполнить ключи API
cp .env.example .env

# 2. Убедиться что Claude Code CLI залогинен
claude --version

# 3. Конкретный файл по номеру
./scripts/run.sh 43
# → output/videos/YYYY-MM-DD__NNN.mp4

# 4. Случайный необработанный файл
./scripts/run.sh

# 5. Бесконечный цикл (все файлы подряд)
./scripts/loop.sh
```

`run.sh` поднимает puppeteer-контейнер (`docker compose up -d`) и запускает pipeline локально (`go run ./cmd/main.go`). Переменные из `.env` подхватываются автоматически.

Результаты:
- `output/` — видео, аудио, сценарии
- `state/` — `processed.json`, `tts_usage.json`

---

## Запуск через Docker (VIA_DOCKER=true)

```bash
# .env: VIA_DOCKER=true
./scripts/run.sh 43
# Эквивалент: docker compose run --rm --build pipeline -num 43
```

Используй этот режим для воспроизводимого окружения или CI. Claude Code CLI в Docker недоступен, поэтому при `VIA_DOCKER=true` нужен `LLM_BACKEND=zai`.

---

## Исправление готового ролика (fix)

Если нужно поправить озвучку/субтитры без перегенерации сценария через LLM:

**1. Найти сценарий**

```
output/scripts/YYYY-MM-DD__NNN.json
```

**2. Отредактировать поля**

| Поле | Что менять |
|---|---|
| `NarrationSSML` | SSML для TTS — ударения (`гор+утина`), паузы (`<break time="300ms"/>`), акценты (`*слово*`) |
| `NarrationText` | Чистый текст субтитров (без SSML) |
| `Segments[].Text` | Текст отдельного сегмента субтитров |

**3. Запустить перегенерацию**

```bash
./scripts/fix.sh 43
```

Скрипт пропускает LLM, берёт отредактированный `NarrationSSML` и заново синтезирует аудио (SaluteSpeech) + рендерит видео (Puppeteer + FFmpeg).

---

## Разработка

```bash
# Unit-тесты (парсер + выбор контента), без внешних зависимостей
go test ./tests/...

# Тест TTS изолированно
go run ./cmd/main.go -test-tts "Привет мир"
# → output/audio/test-tts.wav

# Логи puppeteer
docker compose logs -f puppeteer

# Остановить puppeteer
docker compose down
```

Для быстрой итерации над шаблонами без пересборки контейнера — `docker-compose.override.yml` (не коммитить):
```yaml
services:
  puppeteer:
    volumes:
      - ./puppeteer:/app
      - ./static:/static
```

---

## Переменные окружения

| Переменная | Описание |
|---|---|
| `VIA_DOCKER` | `false` — pipeline локально, `true` — pipeline в Docker |
| `LLM_BACKEND` | `claude-cli` — Claude Code CLI, `zai` — z.ai API |
| `ZAI_API_KEY` | Ключ z.ai API (нужен только при `LLM_BACKEND=zai`) |
| `ZAI_MODEL` | Модель z.ai, по умолчанию `glm-5` |
| `SALUTESPEECH_CLIENT_ID` | SaluteSpeech Client ID |
| `SALUTESPEECH_CLIENT_SECRET` | SaluteSpeech Client Secret |
| `SALUTESPEECH_SCOPE` | `SALUTE_SPEECH_PERS` (физ. лицо) или `SALUTE_SPEECH_CORP` |
| `SALUTESPEECH_VOICE` | `Nec_24000` (нейтральный женский), `Bys_24000` (деловой) |
| `PUPPETEER_URL` | URL puppeteer-сервиса, по умолчанию `http://localhost:3333` |
| `RAW_DIR` | Папка с markdown-файлами, по умолчанию `./raw` |
| `OUTPUT_DIR` | Папка с результатами, по умолчанию `./output` |
| `STATE_DIR` | Папка с состоянием, по умолчанию `./state` |
| `LANG` | Язык нарратива: `ru` (по умолчанию), `en`, `es` |
