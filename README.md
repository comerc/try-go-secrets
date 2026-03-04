# Semarang - YouTube Shorts Production Pipeline for Go Secrets

Automated pipeline for producing daily YouTube Shorts from 286 markdown files containing Go programming secrets.

## Features

- **Daily automated video generation** from markdown files
- **Russian female voice** (Alice from Yandex SpeechKit)
- **Animated code** with syntax highlighting (Termynal + Prism.js)
- **FiraCode font** with ligatures
- **Docker-based** isolation and deployment
- **CLI interface** for manual control

## Usage

```bash
# Generate video for a specific file number (e.g., line-043.md)
./run.sh 43

# Generate video for a random unprocessed file
./run.sh

# Run via Docker Compose
docker-compose run --rm pipeline ./run.sh 43
```

## Project Structure

```
semarang/
├── cmd/main.go              # Entry point
├── pkg/                     # Go packages
│   ├── config/              # Configuration
│   ├── agents/              # AI agents
│   ├── services/            # External APIs
│   ├── models/              # Data models
│   ├── orchestrator/         # Orchestration
│   └── state/              # State management
├── puppeteer/              # Node.js service
│   ├── server.js           # HTTP server
│   └── templates/          # HTML templates
├── static/                 # Static assets
├── raw/                   # Source markdown files
├── output/                # Generated files
├── state/                 # Production state
└── scripts/run.sh         # CLI entry point
```

## Configuration

Copy `.env.example` to `.env` and add your API keys:

```bash
cp .env.example .env
# Edit .env with your ZAI_API_KEY and YANDEX_API_KEY
```

## Requirements

- Go 1.21+
- Node.js 18+
- FFmpeg
- Docker & Docker Compose (optional)

## License

MIT
