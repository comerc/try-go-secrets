const express = require('express');
const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs');
const { execSync } = require('child_process');

const app = express();
app.use(express.json({ limit: '10mb' }));
app.use('/static', express.static(path.join(__dirname, '..', 'static')));

const PORT = process.env.PORT || 3000;
const templatePath = path.join(__dirname, 'templates', 'terminal.html');

// GET /terminal?lang=go — отдаёт HTML шаблон через HTTP
// Нужно чтобы Puppeteer загружал страницу через URL и /static/ ресурсы резолвились
app.get('/terminal', (req, res) => {
  const lang = req.query.lang || 'go';
  const html = fs.readFileSync(templatePath, 'utf8');
  res.send(html);
});

// POST /render — генерирует MP4 из переданного кода
// Body: { code, lang, slug, outputPath, width, height, fps, audioDuration, narration }
app.post('/render', async (req, res) => {
  const {
    code,
    lang = 'go',
    slug,
    outputPath,
    width = 1080,
    height = 1920,
    fps = 30,
    audioDuration = 50,
    subtitleWords = [],
  } = req.body;

  if (!code || !outputPath) {
    return res.status(400).json({ error: 'code и outputPath обязательны' });
  }

  const screenshotsDir = path.join(path.dirname(outputPath), 'frames', slug || 'tmp');
  fs.mkdirSync(screenshotsDir, { recursive: true });

  let browser;
  try {
    const executablePath = process.env.PUPPETEER_EXECUTABLE_PATH || puppeteer.executablePath();
    browser = await puppeteer.launch({
      headless: true,
      executablePath,
      args: [
        '--no-sandbox',
        '--disable-setuid-sandbox',
        '--disable-dev-shm-usage',
        `--window-size=${width},${height}`,
      ],
    });

    const page = await browser.newPage();
    await page.setViewport({ width, height });

    // Загружаем через HTTP — так /static/ ресурсы (highlight.js, шрифты) резолвятся корректно
    await page.goto(`http://localhost:${PORT}/terminal?lang=${encodeURIComponent(lang)}`, {
      waitUntil: 'networkidle0',
    });

    // Анимируем посимвольный ввод кода и делаем скриншоты кадров
    const totalFrames = Math.ceil(audioDuration * fps);
    const totalChars = code.length;
    const charsPerFrame = Math.max(1, totalChars / (totalFrames * 0.7)); // анимация на 70% времени

    // Слова для субтитров (display-версия, без +)
    const subtitleDisplayWords = subtitleWords.map(w => w.word);

    // Для каждого кадра заранее вычисляем индекс текущего слова
    // по временным меткам startSec каждого слова
    const frameWordIdx = new Int32Array(totalFrames).fill(-1);
    for (let wi = 0; wi < subtitleWords.length; wi++) {
      const startFrame = Math.floor(subtitleWords[wi].startSec * fps);
      const endFrame = wi + 1 < subtitleWords.length
        ? Math.floor(subtitleWords[wi + 1].startSec * fps)
        : totalFrames;
      for (let f = Math.max(0, startFrame); f < Math.min(endFrame, totalFrames); f++) {
        frameWordIdx[f] = wi;
      }
    }

    let charsShown = 0;
    for (let frame = 0; frame < totalFrames; frame++) {
      if (frame < totalFrames * 0.7) {
        charsShown = Math.min(totalChars, Math.round(frame * charsPerFrame));
      } else {
        charsShown = totalChars; // остаток — показываем весь код
      }

      const partial = code.slice(0, charsShown);
      const currentWordIdx = frameWordIdx[frame];

      await page.evaluate((text, language, words, wordIdx) => {
        if (typeof updateCode === 'function') updateCode(text, language);
        if (typeof updateSubtitle === 'function') updateSubtitle(words, wordIdx);
      }, partial, lang, subtitleDisplayWords, currentWordIdx);

      const framePath = path.join(screenshotsDir, `frame-${String(frame).padStart(6, '0')}.png`);
      await page.screenshot({ path: framePath, type: 'png' });
    }

    await browser.close();

    // Сборка видео из кадров через FFmpeg
    fs.mkdirSync(path.dirname(outputPath), { recursive: true });
    const ffmpegCmd = [
      'ffmpeg -y',
      `-framerate ${fps}`,
      `-i "${path.join(screenshotsDir, 'frame-%06d.png')}"`,
      '-c:v libx264',
      '-pix_fmt yuv420p',
      '-preset fast',
      `-s ${width}x${height}`,
      `"${outputPath}"`,
    ].join(' ');

    execSync(ffmpegCmd, { stdio: 'pipe' });

    // Чистим кадры
    fs.rmSync(screenshotsDir, { recursive: true, force: true });

    res.json({ success: true, outputPath });
  } catch (err) {
    if (browser) await browser.close().catch(() => {});
    console.error('Ошибка рендеринга:', err);
    res.status(500).json({
      error: err && err.message ? err.message : 'unknown puppeteer error',
    });
  }
});

// GET /health
app.get('/health', (_, res) => res.json({ status: 'ok' }));

app.listen(PORT, () => {
  console.log(`Puppeteer сервис запущен на порту ${PORT}`);
});
