# Миграция TTS: Yandex SpeechKit → SaluteSpeech

## Контекст

Текущий проект использует Yandex SpeechKit с голосом "Алиса" для синтеза речи. Голос Алисы через API доступен только в рамках дорогой услуги Brand Voice с юридическим согласованием. Для озвучки обучающего контента по Golang выбран **SaluteSpeech** от Сбера:

- Бесплатный лимит ~200 000 символов/месяц
- Качественные русские голоса с хорошей интонацией
- Понимание технических терминов
- Оплата российскими картами

**Цель:** Заменить интеграцию Yandex SpeechKit на SaluteSpeech с минимальными изменениями в архитектуре.

---

## Файлы для изменения

| Файл | Изменения |
|------|-----------|
| `pkg/services/tts_service.go` | Полная переработка под SaluteSpeech API |
| `pkg/config/config.go` | Замена Yandex-переменных на SaluteSpeech |
| `.env.example` | Обновление переменных окружения |
| `pkg/orchestrator/orchestrator.go` | Обновление инициализации TTS-сервиса |

---

## Различия API

| Параметр | Yandex SpeechKit | SaluteSpeech |
|----------|------------------|--------------|
| Аутентификация | API-Key + Folder ID | OAuth токен (Bearer) |
| Endpoint | `https://stt.api.cloud.yandex.net/speech/v1/tts:synthesize` | `https://smartspeech.sber.ru/api/v1/synthesis` |
| Формат запроса | URL-encoded параметры | JSON body |
| Голоса | alice, filipp | Наталья, Александр, Марина, Александр_С, Татьяна, Елена, Анна |
| Формат аудио | MP3 | MP3, WAV, OPUS |
| Лимит текста | 5000 символов | ~5000 символов |

---

## План реализации

### 1. Обновление конфигурации (`pkg/config/config.go`)

**Удалить:**
```go
YandexAPIKey     string
YandexFolderID   string
YandexTTSURL     string
YandexVoice      string
YandexLanguage   string
```

**Добавить:**
```go
// SaluteSpeech TTS
SaluteSpeechToken    string  // OAuth токен
SaluteSpeechURL      string  // API endpoint
SaluteSpeechVoice    string  // Имя голоса
```

### 2. Обновление TTS-сервиса (`pkg/services/tts_service.go`)

**Изменения в структуре:**
```go
type TTSService struct {
    token      string           // Bearer токен вместо API-Key
    baseURL    string
    voice      string
    httpClient *http.Client
    usageMgr   *state.TTSUsageManager
    maxChars   int
}
```

**Изменения в Synthesize():**
- Формирование JSON-тела запроса вместо URL-параметров
- Заголовок `Authorization: Bearer <token>` вместо `Api-Key`
- Обработка ответа SaluteSpeech (бинарные данные или JSON с ошибкой)

**Пример запроса SaluteSpeech:**
```go
type synthesisRequest struct {
    Text   string `json:"text"`
    Voice  string `json:"voice_name"`
    Format string `json:"audio_format"`
}

body, _ := json.Marshal(synthesisRequest{
    Text:   text,
    Voice:  s.voice,
    Format: "mp3",
})

req, _ := http.NewRequest("POST", s.baseURL, bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
req.Header.Set("Authorization", "Bearer "+s.token)
```

### 3. Обновление переменных окружения (`.env.example`)

**Удалить:**
```bash
YANDEX_API_KEY=...
YANDEX_FOLDER_ID=...
YANDEX_TTS_URL=...
YANDEX_VOICE=alice
YANDEX_LANGUAGE=ru-RU
```

**Добавить:**
```bash
# SaluteSpeech TTS Configuration
SALUTESPEECH_TOKEN=your_oauth_token_here
SALUTESPEECH_URL=https://smartspeech.sber.ru/api/v1/synthesis
SALUTESPEECH_VOICE=Наталья

# TTS Settings (сохраняем)
TTS_MAX_CHARS_PER_DAY=200000
TTS_SPEECH_RATE_CHARS_PER_SEC=8.0
```

### 4. Обновление оркестратора (`pkg/orchestrator/orchestrator.go`)

Изменить инициализацию TTS-сервиса (строки 55-63):
```go
ttsService := services.NewTTSService(
    cfg.SaluteSpeechToken,
    cfg.SaluteSpeechURL,
    cfg.SaluteSpeechVoice,
    ttsUsageMgr,
    cfg.TTSMaxCharsPerDay,
)
```

---

## Рекомендуемые голоса SaluteSpeech

Для обучающего контента по Golang:

| Голос | Характеристика |
|-------|---------------|
| **Наталья** | Женский, спокойный, лекторский — ВЫБРАН |
| Татьяна | Женский, мягкий |
| Елена | Женский, энергичный |

---

## Получение токена SaluteSpeech

1. Перейти на https://cloud.sber.ru или https://salutespeech.sber.ru
2. Зарегистрироваться/авторизоваться
3. Создать проект в консоли
4. Получить API-ключ или OAuth-токен в настройках проекта
5. Добавить токен в `.env`:
   ```bash
   SALUTESPEECH_TOKEN=<ваш_токен>
   ```

**Бесплатный лимит:** ~200 000 символов/месяц для физических лиц.

---

## Порядок выполнения

1. **[ ]** Обновить `pkg/config/config.go` — заменить Yandex-поля на SaluteSpeech
2. **[ ]** Переписать `pkg/services/tts_service.go` под SaluteSpeech API
3. **[ ]** Обновить `.env.example` с новыми переменными
4. **[ ]** Адаптировать `pkg/orchestrator/orchestrator.go` — новая сигнатура NewTTSService
5. **[ ]** Тестирование: запустить пайплайн с тестовым файлом
6. **[ ]** Проверить качество аудио и корректность технических терминов

---

## Верификация

После миграции запустить:
```bash
./run.sh 1  # или любой доступный номер файла
```

Проверить:
- [ ] Аудиофайл создаётся в `output/audio/`
- [ ] Качество голоса приемлемое
- [ ] Технические термины (горутина, мьютекс, канал) произносятся корректно
- [ ] Видео генерируется успешно
- [ ] Трекинг использования TTS работает

---

# Оригинальный план проекта

> Ниже приведён оригинальный план создания пайплайна.

---

## Контекст (оригинал)

Создать автоматизированный пайплайн для ежедневного производства YouTube Shorts (< 60 секунд) из 286 существующих markdown-файлов с секретами Go.

**Ключевые требования:**
- 1 видео в день из 286 исходных файлов (~год контента)
- Русский голос для озвучки технического контента (теперь SaluteSpeech)
- Анимированный код с подсветкой синтаксиса (Termynal + Prism.js + Puppeteer)
- Docker-compose для изоляции
- Только генерация MP4 (без авто-публикации)
- **Основной код на Golang**

---

## Структура проекта

```
semarang/
├── raw/                                    # Вход: 286 markdown-файлов
│   └── *.md                               # Исходный контент (уже существует)
├── cmd/                                    # Точка входа (Go)
│   └── main.go                            # Основной executable
├── pkg/                                    # Основной код пайплайна (Go)
│   ├── config/                            # Управление конфигурацией
│   │   └── config.go
│   ├── orchestrator/                      # Оркестрация
│   │   └── orchestrator.go
│   ├── agents/                            # AI агенты
│   │   ├── content_selector.go             # Выбор контента по номеру
│   │   ├── script_writer.go                # Создание сценария видео
│   │   ├── code_extractor.go               # Извлечение блоков кода
│   │   ├── video_generator.go              # Генерация видео
│   │   └── quality_checker.go              # Валидация вывода
│   ├── services/                          # Интеграция с внешними API
│   │   ├── llm_service.go                 # Клиент z.ai API
│   │   ├── tts_service.go                 # Клиент Yandex SpeechKit
│   │   ├── video_service.go               # Обёртка Termynal/Puppeteer
│   │   └── content_parser.go              # Парсер markdown
│   ├── models/                            # Модели данных
│   │   ├── content.go                     # Модель контента
│   │   ├── script.go                      # Модель сценария
│   │   └── video_spec.go                  # Спецификация видео
│   └── state/                             # Управление состоянием
│       ├── processed.go                    # Отслеживание обработанных файлов
│       ├── queue.go                        # Очередь контента
│       └── tts_usage.go                   # Отслеживание использования TTS
├── puppeteer/                              # Node.js сервис для Puppeteer
│   ├── package.json
│   ├── server.js                          # HTTP сервер для генерации видео
│   └── templates/                         # HTML-шаблоны
│       ├── terminal.html                  # Шаблон для анимации кода
│       └── diagram.html                   # Шаблон для mermaid-диаграмм
├── static/                                 # Статические ресурсы
│   ├── css/
│   │   └── terminal.css                   # Стили терминала
│   │   └── fira-code.css                  # FiraCode шрифт с лигатурами
│   └── js/
│       ├── termynal.js                    # Библиотека анимации терминала
│       └── prism.js                       # Подсветка синтаксиса
├── docker/                                 # Конфигурации Docker
│   ├── Dockerfile                         # Основной контейнер (Go)
│   ├── Dockerfile.puppeteer               # Контейнер Puppeteer (Node.js)
│   └── docker-compose.yml                # Оркестрация
├── output/                                 # Генерируемые файлы (gitignored)
│   ├── scripts/                           # Сгенерированные сценарии видео
│   ├── audio/                             # Сгенерированные TTS аудиофайлы
│   └── videos/                            # Финальные MP4 файлы
│   └── logs/                              # Логи производства
├── state/                                  # Состояние производства (в git)
│   ├── processed.json                     # Отслеживание обработанных файлов
│   └── tts_usage.json                     # Отслеживание использования TTS
├── scripts/                                # Вспомогательные скрипты
│   └── run.sh                            # Запуск пайплайна (./run.sh [number])
├── tests/                                  # Набор тестов (Go)
│   └── ...
├── .env.example                            # Шаблон переменных окружения
├── .env                                    # API ключи (gitignored)
├── go.mod                                  # Модули Go
├── go.sum
├── Dockerfile                              # Основной Dockerfile
└── docker-compose.yml                      # Оркестрация Docker
```

---

## Архитектура

### Поток данных

```
┌─────────────────────────────────────────────────────────────────┐
│                         ЗАПУСК: ./run.sh [NUM]                 │
│  • NUM: номер файла (например, 43 для Go_Secret__line-043.md)   │
│  • Без NUM: случайный необработанный файл                      │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────────┐
│                     ContentSelector                              │
│  • Парсит raw/ для файлов по шаблону *-line-<NUM>.md         │
│  • Проверяет state/processed.json                               │
│  • Возвращает выбранный файл или ошибку (если уже обработан)   │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼ (путь к выбранному файлу)
┌─────────────────────────────────────────────────────────────────┐
│                      ScriptWriter                               │
│  • Парсит markdown-файл                                        │
│  • Создаёт сценарий < 60с (через z.ai API)                     │
│  • Рассчитывает тайминг сегментов                              │
│  • Сохраняет сценарий в output/scripts/YYYY-MM-DD.json         │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼ (объект сценария)
┌─────────────────────────────────────────────────────────────────┐
│                     CodeExtractor                               │
│  • Извлекает блоки кода (```go)                               │
│  • Извлекает mermaid-диаграммы (```mermaid)                    │
│  • Форматирует для отображения                                 │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼ (код/диаграмма)
┌─────────────────────────────────────────────────────────────────┐
│                     VideoGenerator                              │
│  • Вызывает Yandex SpeechKit для голосовых сегментов          │
│  • Сохраняет аудио в output/audio/                             │
│  • Отправляет запрос в Puppeteer-сервис (Node.js)              │
│  • Получает видео с анимацией кода                             │
│  • Объединяет сегменты через FFmpeg                             │
│  • Сохраняет MP4 в output/videos/                               │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼ (путь к видеофайлу)
┌─────────────────────────────────────────────────────────────────┐
│                     QualityChecker                              │
│  • Проверяет длительность < 60с                                  │
│  • Проверяет качество аудио                                    │
│  • Верифицирует формат MP4                                     │
│  • При успехе: обновляет state/processed.json                  │
│  • При неудаче: помечает для пересмотра                        │
└──────────────────────────┬──────────────────────────────────────┘
                           │
                           ▼
                    ┌──────────┐
                    │   КОНЕЦ  │
                    │  Готово  │
                    └──────────┘
```

---

## Технологический стек

| Компонент | Технология | Назначение |
|-----------|-----------|------------|
| Основной язык | Golang | Логика пайплайна, API интеграции |
| AI Оркестрация | Прямые вызовы API (CrewAI не критичен) | Логика агентов |
| LLM | z.ai API (GLM Coding Pro) | Генерация сценариев |
| TTS | Yandex SpeechKit (бесплатный тариф) | Русский женский голос (Алиса), 2000 символов/день |
| Анимация кода | Termynal.js + Prism.js | Анимация набора кода в стиле терминала |
| Шрифт кода | FiraCode с лигатурами | Красивое отображение кода |
| Захват видео | Puppeteer (Node.js) | Автоматизация браузера для записи видео |
| Обработка видео | FFmpeg | Объединение аудио/видео и кодирование |
| Контейнеризация | Docker + docker-compose | Изоляция и деплой |
| Парсинг контента | Go + regex | Парсинг markdown (```go, ```mermaid, ```old блоки) |
| Управление состоянием | JSON-файлы | Отслеживание обработанного контента |

---

## Definition of Done (DOD)

Производство видео считается **ЗАВЕРШЁННЫМ**, когда:

### Требования к видео
- [ ] Длительность строго меньше 60 секунд (цель: 50-55с)
- [ ] Формат файла MP4 с видеокодеком H.264 и аудиокодеком AAC
- [ ] Разрешение минимум 1080x1920 (вертикальное для YouTube Shorts)
- [ ] Размер файла менее 100MB
- [ ] Видео воспроизводится в стандартных видеоплеерах

### Требования к аудио
- [ ] Голос русский женский (Алиса), мягкий, ~30 лет
- [ ] Аудио чистое и разборчивое
- [ ] Без фонового шума и артефактов
- [ ] Темп речи естественный (не торопливый, не медленный)
- [ ] Аудио синхронизировано с визуальным контентом

### Требования к контенту
- [ ] Сценарий раскрывает главный секрет Go
- [ ] Объяснение на естественном разговорном русском
- [ ] Пример кода (если есть) чётко виден с шрифтом FiraCode и лигатурами
- [ ] Диаграммы (если есть) отображаются корректно
- [ ] Длина контента укладывается в лимит TTS Yandex (2000 символов/день)

### Требования к состоянию
- [ ] Исходный markdown-файл отмечен как обработанный в `state/processed.json`
- [ ] Путь к видео записан
- [ ] Дата записана
- [ ] Использование TTS отслеживается в `state/tts_usage.json`

### Требования к выводу
- [ ] MP4-файл существует в `output/videos/`
- [ ] Имя файла соответствует формату: `YYYY-MM-DD-{slug}.mp4`
- [ ] JSON-сценарий существует в `output/scripts/`
- [ ] Аудиосегменты существуют в `output/audio/`
- [ ] Лог производства существует в `output/logs/`

### Требования к качеству
- [ ] Код читаем в видео (FiraCode шрифт, лигатуры, правильный размер, контраст)
- [ ] Анимация плавная (без подёргиваний)
- [ ] Без визуальных глюков и артефактов
- [ ] Громкость аудио нормализована

### Требования к CLI
- [ ] `./run.sh <NUM>` генерирует видео для конкретного номера
- [ ] `./run.sh` выбирает случайный необработанный файл
- [ ] Выводит понятный прогресс выполнения
- [ ] Обрабатывает ошибки корректно (например, если файл уже обработан)

---

## Критические файлы для создания

| Файл | Назначение |
|------|------------|
| `scripts/run.sh` | CLI скрипт для запуска пайплайна |
| `cmd/main.go` | Точка входа приложения |
| `pkg/config/config.go` | Управление конфигурацией (API ключи, пути, настройки) |
| `pkg/orchestrator/orchestrator.go` | Основная логика оркестрации |
| `pkg/services/tts_service.go` | Интеграция Yandex SpeechKit |
| `pkg/services/llm_service.go` | Интеграция z.ai API |
| `pkg/services/video_service.go` | Генерация видео через Puppeteer + FFmpeg |
| `pkg/services/content_parser.go` | Парсер markdown для Go-кода, mermaid-диаграмм |
| `pkg/agents/script_writer.go` | Генерация сценариев через z.ai API |
| `pkg/agents/content_selector.go` | Выбор контента по номеру или случайно |
| `puppeteer/server.js` | Node.js сервер для Puppeteer |
| `puppeteer/templates/terminal.html` | HTML-шаблон с FiraCode |
| `docker-compose.yml` | Оркестрация контейнеров |
| `Dockerfile` | Контейнер Go с FFmpeg |
| `Dockerfile.puppeteer` | Контейнер Node.js с Chrome + Puppeteer |
| `.env.example` | Шаблон переменных окружения |
| `go.mod` | Зависимости Go |

---

## Этапы реализации

### Этап 1: Фундамент
1. Создание структуры каталогов проекта
2. Инициализация Go модуля (`go mod init`)
3. Создание `scripts/run.sh` с парсингом аргументов (номер файла или случайный)
4. Реализация `pkg/config/config.go` с управлением конфигурацией
5. Создание `.env.example` с плейсхолдерами API ключей

### Этап 2: Парсинг контента
1. Реализация `pkg/services/content_parser.go` для парсинга markdown-файлов
2. Извлечение: текст объяснения, блоки кода Go (```go), mermaid-диаграммы (```mermaid), старый контент (```old)
3. Создание моделей данных в `pkg/models/`
4. Добавление тестов для парсинга контента

### Этап 3: Выбор контента
1. Реализация `pkg/agents/content_selector.go` - выбор по номеру или случайно из необработанных
2. Реализация `pkg/state/processed.go` для чтения/записи `state/processed.json`
3. Парсинг имён файлов для извлечения номера (например, `Go_Secret__line-043.md` → 43)

### Этап 4: Интеграция API
1. Реализация `pkg/services/llm_service.go` для z.ai API
2. Реализация `pkg/services/tts_service.go` для Yandex SpeechKit
3. Добавление отслеживания использования TTS в `state/tts_usage.json`
4. Тестирование API интеграций

### Этап 5: Генерация сценария
1. Реализация `pkg/agents/script_writer.go` - создание сценария < 60с через z.ai API
2. Расчёт времени сегментов (темп русской речи: ~8 символов/сек)
3. Сохранение сценариев в `output/scripts/`

### Этап 6: Puppeteer сервис (Node.js)
1. Настройка Node.js проекта в `puppeteer/`
2. Реализация `puppeteer/server.js` - HTTP сервер для генерации видео
3. Создание HTML-шаблона `terminal.html` с Termynal.js + Prism.js + FiraCode
4. Настройка FFmpeg для захвата видео из браузера

### Этап 7: Генерация видео
1. Реализация `pkg/services/video_service.go` - вызов Puppeteer сервиса
2. Интеграция FFmpeg для объединения аудио/видео
3. Реализация `pkg/agents/video_generator.go`
4. Сохранение видео в `output/videos/`

### Этап 8: Контроль качества
1. Реализация `pkg/agents/quality_checker.go`
2. Проверка длительности < 60с, формата, качества аудио
3. Обновление `state/processed.json` при успехе

### Этап 9: Оркестрация
1. Реализация `pkg/orchestrator/orchestrator.go` - связывает все компоненты
2. Реализация `cmd/main.go` - точка входа
3. Добавление логирования и обработки ошибок

### Этап 10: Docker
1. Создание `Dockerfile` для Go контейнера
2. Создание `Dockerfile.puppeteer` для Node.js контейнера
3. Создание `docker-compose.yml`
4. Тестирование в Docker

### Этап 11: Тестирование и оптимизация
1. Создание набора тестов (Go)
2. Интеграционное тестирование end-to-end
3. Оптимизация производительности
4. Документация

---

## Детали реализации

### Выбор файла по номеру

Файлы в `raw/` имеют формат: `<name>-line-<NUM>.md` (например, `defer_error_propagation__line-202.md`).

Логика выбора:
1. Если указан NUM: ищет файл `*-line-<NUM>.md` (с доп. нулями: 43 → `*-line-043.md`)
2. Если NUM не указан: выбирает случайный файл, которого нет в `state/processed.json`
3. Возвращает ошибку, если файл не найден или уже обработан

### Шрифт FiraCode

FiraCode - монопространственный шрифт с программными лигатурами (например, `!=` → `≠`, `=>` → `⇒`).

Интеграция:
- Скачивание шрифта из CDN или локального файла
- Подключение CSS с `@font-face`
- Включение лигатур через `font-feature-settings: "calt" 1, "ss01" 1`

### Пример шаблона HTML с FiraCode

```html
<!DOCTYPE html>
<html>
<head>
  <link rel="stylesheet" href="/static/css/prism.css">
  <link rel="stylesheet" href="/static/css/terminal.css">
  <link rel="stylesheet" href="/static/css/fira-code.css">
  <script src="/static/js/prism.js"></script>
  <script src="/static/js/termynal.js"></script>
</head>
<body>
  <div class="termynal" data-termynal>
    <span class="prompt">$</span>
    <span class="code">
      <code class="language-go" style="font-family: 'Fira Code', monospace;">
        {{.Code}}
      </code>
    </span>
  </div>
</body>
</html>
```
