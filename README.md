# ComTrade Viewer

Web-based viewer for ComTrade (IEEE C37.111) waveform files with interactive visualization.

## Features

- 📤 Upload and parse `.cfg` + `.dat` ComTrade file pairs
- 📊 Interactive waveform display using ECharts
- 🔍 Zoom, pan, and explore time-series data
- 📝 Channel selection and multi-trace visualization
- 💾 Local file storage with metadata caching
- 🚀 High-performance rendering for large datasets

## Tech Stack

- **Frontend**: Vue 3, Vite, TypeScript, Pinia, ECharts
- **Backend**: Go 1.21+, Gin framework
- **Storage**: Local filesystem (upgradable to S3/DB)

## Prerequisites

- Go 1.21+ ([download](https://go.dev/dl/))
- Node.js 20+ ([download](https://nodejs.org/))

## Quick Start

### Option 1: Use the convenience script

```bash
./dev.sh
```

This starts both backend (`:8080`) and frontend (`:5173`) in one command.

### Option 2: Manual startup

**Terminal 1 - Backend:**

```bash
cd backend
go mod tidy
go run .
```

**Terminal 2 - Frontend:**

```bash
cd frontend
npm install
npm run dev
```

Then open http://localhost:5173 in your browser.

## Usage

1. Click **"Import Dataset"** and select a `.cfg` and `.dat` file pair
   - Try the example files in `examples/test.cfg` and `examples/test.dat`
2. Your dataset appears in the **Datasets** list
3. Click a dataset to load its metadata
4. Check channels in the sidebar to visualize them
5. Use the chart controls:
   - **Mouse wheel** to zoom time axis
   - **Drag** to pan
   - **Slider** below chart for navigation

## Project Structure

```
comTradeViewer/
├── backend/
│   ├── main.go          # Go API server (Gin)
│   ├── go.mod
│   └── data/            # Uploaded datasets (runtime)
├── frontend/
│   ├── src/
│   │   ├── components/  # Vue components
│   │   ├── stores/      # Pinia state management
│   │   ├── api.ts       # API client
│   │   └── main.ts
│   ├── vite.config.ts
│   └── package.json
├── docs/
│   └── design.md        # Full design document
├── examples/
│   ├── test.cfg         # Sample ComTrade config
│   └── test.dat         # Sample ComTrade data
└── dev.sh               # Development startup script
```

## API Endpoints

- `POST /api/datasets/import` - Upload cfg + dat files
- `GET /api/datasets` - List all datasets
- `GET /api/datasets/:id/metadata` - Get parsed metadata
- `GET /api/datasets/:id/waveforms?channels=A1,A2&start=0&end=500` - Fetch waveform data
- `GET/POST/DELETE /api/datasets/:id/annotations` - Manage annotations

## Development Roadmap

See [docs/design.md](docs/design.md) for the complete design specification.

**M1 - Core (Current):**

- ✅ Upload & parse basic ComTrade
- ✅ Metadata extraction
- ✅ Interactive chart with ECharts
- 🔄 Real `.dat` binary/ASCII parsing (placeholder synthetic data currently)

**M2 - Performance:**

- ⬜ Chunk-based indexing for large files
- ⬜ LTTB downsampling
- ⬜ SSE/WebSocket streaming

**M3 - Features:**

- ⬜ Annotations & markers
- ⬜ Export to PNG/CSV
- ⬜ Multi-dataset comparison

**M4 - Production:**

- ⬜ Docker Compose deployment
- ⬜ Authentication
- ⬜ S3 storage backend

## Contributing

This is a demonstration project. For production use, consider:

- Robust ComTrade parser (handle all variants: 1991/1999, ASCII/BINARY/BINARY32)
- Input validation and error handling
- Rate limiting and security hardening
- Test coverage and CI/CD

## License

MIT
