# Code Review Report — Go Secrets Pipeline

## Критические

### 1. Shell injection в `createAudioOnlyVideo`
**Файл:** [pkg/agents/video_generator.go](pkg/agents/video_generator.go#L158)
`audioPath` и `outputPath` вставляются в shell-строку через `fmt.Sprintf` и выполняются через `exec.Command("sh", "-c", cmd)`. Если путь содержит кавычки или спецсимволы — команда сломается или будет скомпрометирована.
**Фикс:** заменить на `exec.Command("ffmpeg", "-y", "-f", "lavfi", ..., audioPath, outputPath)` без shell.

### 2. Command injection в Puppeteer FFmpeg
**Файл:** [puppeteer/server.js](puppeteer/server.js#L110)
FFmpeg-команда собирается конкатенацией строк и передаётся в `execSync()`. Если `outputPath` или `slug` содержат спецсимволы — возможна инъекция команды.
**Фикс:** заменить `execSync(строка)` на `spawnSync('ffmpeg', ['-framerate', fps, '-i', ...])`.

### 3. Деление на ноль
**Файл:** [pkg/agents/video_generator.go](pkg/agents/video_generator.go#L53)
```go
timeScale := actualAudioDur / script.TotalSeconds
```
Если `TotalSeconds == 0` — результат `+Inf`, все тайминги субтитров сломаются.
**Фикс:** проверить `script.TotalSeconds > 0` перед делением.

---

## Высокие

### 4. Path traversal в Puppeteer
**Файл:** [puppeteer/server.js](puppeteer/server.js#L29)
`outputPath` из тела запроса не проверяется. Запрос с `outputPath: "../../etc/cron.d/evil"` запишет файл за пределами output-директории.
**Фикс:** проверять что `outputPath` начинается с разрешённого префикса.

### 5. Временные кадры не удаляются при ошибке
**Файл:** [puppeteer/server.js](puppeteer/server.js#L124)
`fs.rmSync(screenshotsDir)` вызывается только при успехе. При любой ошибке тысячи PNG-кадров остаются на диске.
**Фикс:** перенести в блок `finally`.

### 6. Нет backoff в `loop.sh`
**Файл:** [scripts/loop.sh](scripts/loop.sh#L8)
При постоянных ошибках скрипт крутится без паузы и заполняет диск логами.
**Фикс:** увеличивать задержку при повторных сбоях (exponential backoff).

### 7. `InsecureSkipVerify: true` в TLS-клиенте
**Файл:** [pkg/services/tts_service.go](pkg/services/tts_service.go#L97)
Отключает проверку сертификата. Комментарий объясняет причину (российский CA), но это уязвимость к MITM.
**Фикс:** добавить корневой сертификат Сбера в пул вместо отключения проверки.

---

## Средние

### 8. Игнорируются ошибки чтения тела ответа
**Файлы:** [pkg/services/video_service.go](pkg/services/video_service.go#L86), [pkg/services/tts_service.go](pkg/services/tts_service.go#L149)
```go
body, _ := io.ReadAll(resp.Body)
```
Если чтение не удастся — сообщение об ошибке API потеряется.

### 9. Линейный поиск в `IsProcessed()`
**Файл:** [pkg/state/processed.go](pkg/state/processed.go)
При каждом старте итерируется по всему списку. При 286 файлах — незначительно, но стоит заменить на `map[int]bool`.

### 10. `os.Remove(rawVideoPath)` без обработки ошибки
**Файл:** [pkg/agents/video_generator.go](pkg/agents/video_generator.go#L100)
Если удаление не удалось — ошибка молча игнорируется, промежуточный файл остаётся.

### 11. `lang` не валидируется в Puppeteer
**Файл:** [puppeteer/server.js](puppeteer/server.js#L27)
Передаётся напрямую в `hljs.highlight()`. Невалидный lang не сломает процесс (есть try/catch), но это бесполезный запрос к highlight.js.

### 12. Нет проверки границ `frameWordIdx`
**Файл:** [puppeteer/server.js](puppeteer/server.js#L75)
Если `subtitleWords[wi].startSec * fps` выходит за пределы `totalFrames` — цикл корректно ограничивается `Math.min`, но крайний случай `startSec > audioDuration` даст слово, которое никогда не покажется.

---

## Низкие / качество кода

### 13. Magic numbers
| Значение | Место | Смысл |
|----------|-------|-------|
| `8.5` | script_writer.go:20 | символов в секунду |
| `0.7` | server.js:68 | доля времени на анимацию |
| `29` | tts_service.go:212 | минуты кэша токена |
| `55.0` | script_writer.go:21 | целевая длина в секундах |

### 14. `strings.Split(path, "/")` вместо `filepath.Base`
**Файл:** [pkg/agents/content_selector.go](pkg/agents/content_selector.go)
Ручной парсинг пути ломается на Windows и выглядит хрупко.
**Фикс:** `filepath.Base(filePath)`.

### 15. Отсутствует `context.Context`
Все HTTP-вызовы (LLM, TTS, Puppeteer) не принимают контекст — нет способа отменить операцию или выставить внешний дедлайн.

### 16. Нет структурированного логирования
Используется `fmt.Printf` вместо `log/slog`. Нет уровней (debug/info/warn/error), нет структурированных полей.

### 17. Валидация конфига неполная
**Файл:** [pkg/config/config.go](pkg/config/config.go#L69)
Проверяются только 3 поля. Не проверяются: `VideoWidth/Height/FPS > 0`, корректность URL, доступность директорий.

### 18. Нет очереди в Puppeteer
**Файл:** [puppeteer/server.js](puppeteer/server.js#L24)
Параллельные запросы запустят несколько браузеров одновременно. При этом каждый рендер потребляет ~1-2 GB RAM.
Текущий пайплайн последовательный, поэтому не критично, но при масштабировании станет проблемой.

---

## Сводка

| Приоритет | Кол-во |
|-----------|--------|
| Критические | 3 |
| Высокие | 4 |
| Средние | 5 |
| Низкие | 6 |

**Самое важное к исправлению:** пп. 1–3 (injection + деление на ноль) и п. 5 (утечка кадров на диск).
