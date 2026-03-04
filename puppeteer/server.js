const express = require('express');
const puppeteer = require('puppeteer');
const path = require('path');
const fs = require('fs').promises;
const { exec } = require('child_process');
const { promisify } = require('util');
const execAsync = promisify(exec);

const app = express();
const PORT = process.env.PORT || 3000;
const OUTPUT_DIR = process.env.OUTPUT_DIR || '/app/output/videos';
const TEMPLATE_DIR = path.join(__dirname, 'templates');
const STATIC_DIR = path.join(__dirname, '../static');

// Ensure output directory exists
fs.mkdir(OUTPUT_DIR, { recursive: true }).catch(console.error);

// Middleware
app.use(express.json({ limit: '10mb' }));
app.use('/static', express.static(STATIC_DIR));

// Health check
app.get('/health', (req, res) => {
  res.json({ status: 'ok', service: 'puppeteer' });
});

// Generate video from code
app.post('/generate', async (req, res) => {
  try {
    const { code, output_path, duration_ms, width, height, typing_speed_ms } = req.body;

    if (!code || !output_path) {
      return res.status(400).json({ error: 'Missing required parameters' });
    }

    console.log(`Generating video for: ${output_path}`);
    console.log(`Duration: ${duration_ms}ms, Size: ${width}x${height}`);

    const result = await generateCodeVideo(code, output_path, {
      durationMs: duration_ms || 12000,
      width: width || 1080,
      height: height || 1920,
      typingSpeedMs: typing_speed_ms || 50,
    });

    res.json({ video_path: result.videoPath, duration_ms: result.durationMs });
  } catch (error) {
    console.error('Error generating video:', error);
    res.status(500).json({ error: error.message });
  }
});

// Generate video with audio
app.post('/generate-with-audio', async (req, res) => {
  try {
    const { code, audio_path, output_path, duration_ms, width, height, typing_speed_ms } = req.body;

    if (!code || !output_path) {
      return res.status(400).json({ error: 'Missing required parameters' });
    }

    console.log(`Generating video with audio for: ${output_path}`);

    // First generate the video
    const videoResult = await generateCodeVideo(code, output_path + '.tmp.mp4', {
      durationMs: duration_ms || 12000,
      width: width || 1080,
      height: height || 1920,
      typingSpeedMs: typing_speed_ms || 50,
    });

    // If audio is provided, combine with video
    let finalPath = videoResult.videoPath;
    if (audio_path) {
      finalPath = await combineVideoAudio(videoResult.videoPath, audio_path, output_path);
    } else {
      // Just rename temp file
      await fs.rename(videoResult.videoPath, output_path);
      finalPath = output_path;
    }

    res.json({ video_path: finalPath, duration_ms: videoResult.durationMs });
  } catch (error) {
    console.error('Error generating video with audio:', error);
    res.status(500).json({ error: error.message });
  }
});

// Generate code video
async function generateCodeVideo(code, outputPath, options) {
  let browser;
  try {
    browser = await puppeteer.launch({
      headless: 'new',
      args: [
        '--no-sandbox',
        '--disable-setuid-sandbox',
        '--disable-dev-shm-usage',
        '--disable-gpu',
      ],
    });

    const page = await browser.newPage();

    // Read template
    const templatePath = path.join(TEMPLATE_DIR, 'terminal.html');
    let template = await fs.readFile(templatePath, 'utf-8');

    // Escape special characters for Go code
    const escapedCode = code
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#039;');

    // Replace placeholders
    template = template.replace(/\{\{\.Code\}\}/g, escapedCode);
    template = template.replace(/\{\{\.StartDelay\}\}/g, '100');
    template = template.replace(/\{\{\.TypingSpeedMs\}\}/g, String(options.typingSpeedMs));
    template = template.replace(/\{\{\.LineDelayMs\}\}/g, '0');
    template = template.replace(/\{\{\.Theme\}\}/g, 'dark');

    // Set content
    await page.setContent(template, { waitUntil: 'networkidle0' });

    // Wait for animation to complete
    const animationComplete = new Promise((resolve) => {
      page.on('console', (msg) => {
        if (msg.text() === 'Animation complete') {
          resolve();
        }
      });
    });

    // Wait for timeout or animation completion
    await Promise.race([
      animationComplete,
      new Promise(resolve => setTimeout(resolve, options.durationMs + 2000)),
    ]);

    // Capture video using ffmpeg
    const tempVideoPath = outputPath;

    // Use ffmpeg to capture the video
    await captureVideo(page, tempVideoPath, options);

    await browser.close();

    // Get video duration
    const durationMs = await getVideoDuration(tempVideoPath);

    return { videoPath: tempVideoPath, durationMs };
  } catch (error) {
    if (browser) await browser.close();
    throw error;
  }
}

// Capture video using ffmpeg
async function captureVideo(page, outputPath, options) {
  const width = options.width;
  const height = options.height;
  const duration = (options.durationMs + 1000) / 1000; // Add 1 second buffer

  // Save HTML to temp file for ffmpeg to read
  const tempHtmlPath = `/tmp/video-${Date.now()}.html`;
  const htmlContent = await page.content();
  await fs.writeFile(tempHtmlPath, htmlContent);

  // Use ffmpeg to capture video
  const command = `ffmpeg -y -loop 1 -i "${tempHtmlPath}" -t ${duration} -vf "scale=${width}:${height}:force_original_aspect_ratio=decrease,pad=${width}:${height}:(ow-iw)/2:(oh-ih)/2" -c:v libx264 -tune stillimage -pix_fmt yuv420p -r 30 "${outputPath}"`;

  try {
    await execAsync(command);
    console.log(`Video captured to: ${outputPath}`);
  } catch (error) {
    console.error('FFmpeg error:', error);
    // Try alternative method with xvfb
    await captureVideoWithXvfb(page, outputPath, options);
  } finally {
    // Clean up temp file
    await fs.unlink(tempHtmlPath).catch(() => {});
  }
}

// Alternative capture method with xvfb (for headless environments)
async function captureVideoWithXvfb(page, outputPath, options) {
  console.log('Using xvfb for video capture...');

  const width = options.width;
  const height = options.height;
  const duration = (options.durationMs + 1000) / 1000;

  // Save HTML to temp file
  const tempHtmlPath = `/tmp/video-${Date.now()}.html`;
  const htmlContent = await page.content();
  await fs.writeFile(tempHtmlPath, htmlContent);

  // Use xvfb-run with ffmpeg
  const command = `xvfb-run -a --server-args="-screen 0 ${width}x${height}x24" ffmpeg -y -loop 1 -i "${tempHtmlPath}" -t ${duration} -vf "scale=${width}:${height}" -c:v libx264 -pix_fmt yuv420p -r 30 "${outputPath}"`;

  try {
    await execAsync(command);
  } catch (error) {
    throw new Error(`Failed to capture video: ${error.message}`);
  } finally {
    await fs.unlink(tempHtmlPath).catch(() => {});
  }
}

// Combine video and audio using ffmpeg
async function combineVideoAudio(videoPath, audioPath, outputPath) {
  const duration = await getVideoDuration(videoPath);

  const command = `ffmpeg -y -i "${videoPath}" -i "${audioPath}" -c:v copy -c:a aac -map 0:v:0 -map 1:a:0 -shortest -t ${duration / 1000} "${outputPath}"`;

  try {
    await execAsync(command);
    console.log(`Combined video and audio to: ${outputPath}`);

    // Clean up temp video
    await fs.unlink(videoPath).catch(() => {});

    return outputPath;
  } catch (error) {
    throw new Error(`Failed to combine video and audio: ${error.message}`);
  }
}

// Get video duration using ffprobe
async function getVideoDuration(videoPath) {
  try {
    const command = `ffprobe -v error -show_entries format=duration -of json "${videoPath}"`;
    const { stdout } = await execAsync(command);
    const result = JSON.parse(stdout);
    return Math.round(result.format.duration * 1000);
  } catch (error) {
    console.error('Failed to get video duration:', error);
    return 0;
  }
}

// Start server
app.listen(PORT, '0.0.0.0', () => {
  console.log(`Puppeteer server running on port ${PORT}`);
  console.log(`Output directory: ${OUTPUT_DIR}`);
});
