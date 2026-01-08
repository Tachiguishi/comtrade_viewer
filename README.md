# ComTrade Viewer

This is a simple web-based viewer for ComTrade files, consisting of a Vue 3 frontend and a Go backend.

## Prerequisites

- Go 1.21+
- Node.js 18+

## Quick Start

### 1. Start the Backend

```bash
cd backend
go mod tidy
go run main.go
```

The backend will run on http://localhost:8080.
Data files are stored in `backend/data`.

### 2. Start the Frontend

In a new terminal:

```bash
cd frontend
npm install
npm run dev
```

The frontend will open at http://localhost:5173.

## Usage

1.  Open the web interface.
2.  Use the "New Import" pane to upload a pair of `.cfg` and `.dat` files.
3.  Select the uploaded dataset from the list.
4.  Check channels in the sidebar to view their waveforms.
5.  Use mouse wheel to zoom the chart.

## Development

- **Backend**: `backend/main.go` contains the single-file implementation using Gin.
- **Frontend**: Vue 3 + Vite + Pinia + ECharts. Source in `frontend/src`.
