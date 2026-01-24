# ComTrade Viewer

Web-based viewer for ComTrade (IEEE C37.111) waveform files with interactive visualization.

## Features

- 📤 上传和解析 `.cfg` + `.dat` ComTrade文件
- 📊 使用 ECharts 进行交互式波形显示
- 🔍 缩放、拖拽与窗口导航，查看大规模时序数据
- 📝 通道选择与多曲线叠加
- 💾 本地文件存储（可切换 MinIO），带数据集 LRU 缓存
- ⚙️ 支持 COMTRADE 1991/1999/2013，`ascii`/`binary`/`binary32`/`float32` 数据解析
- 📉 自动 LTTB 下采样（模拟量）与数字量状态变化抽取，提升大数据集渲染性能
- 🔐 基于 JWT 的登录鉴权（默认凭据可配置）

## Tech Stack

- **Frontend**: Vue 3, Vite, TypeScript, Pinia, Naive UI, ECharts, Axios
- **Backend**: Go 1.21+, Gin, JWT (golang-jwt)
- **Storage**: 本地文件系统（默认）或 MinIO（S3 兼容）

## Prerequisites

- Go 1.21+ ([download](https://go.dev/dl/))
- Node.js 20+ ([download](https://nodejs.org/))

## Quick Start

### Option 1: Use the convenience script

```bash
./dev.sh
```

This starts both backend (`:8080`) and frontend (`:5173`) in one command. 默认使用本地存储到 `backend/data`。

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

浏览器中打开 http://localhost:5173.

默认启用鉴权，首次访问会跳转到登录页。

默认登录凭据（可通过环境变量覆盖）：

- 用户名：`AUTH_USERNAME=admin`
- 密码：`AUTH_PASSWORD=admin123`
- JWT 密钥：`AUTH_SECRET=supersecretkey`

## Usage

1. 登录（见上面的默认凭据）。
2. 点击 **“导入数据集”**，选择一对 `.cfg` 与 `.dat` 文件。
   - 可使用示例文件：`backend/test/data/test/cfg` 与 `backend/test/data/test/dat`。
3. 数据集会出现在 **Datasets** 列表中。
4. 选择数据集以加载元数据。
5. 在侧栏勾选模拟量/数字量通道进行可视化。
6. 图表操作：
   - 鼠标滚轮：缩放 X 轴（索引/时间模式可切换）。
   - 拖拽：平移窗口。
   - 下方窗口滑块：快速导航与范围调整。

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
└── dev.sh               # Development startup script

示例数据位于：`backend/test/data/test/`（用于本地上传测试）。
```

## API Endpoints

- `POST /api/auth/login` - 登录获取 JWT（响应同时设置 HttpOnly Cookie）
- `POST /api/datasets/import` - 上传 `.cfg` + `.dat` 文件对（multipart/form-data）
- `GET /api/datasets` - 列出所有数据集
- `GET /api/datasets/:id/metadata` - 解析并返回 CFG 元数据
- `GET /api/datasets/:id/waveforms` - 获取波形数据（支持下采样与时间窗口）
  - 查询参数：
    - `A=1,2,3` 指定模拟量通道编号集合（从 1 开始）
    - `D=1,2` 指定数字量通道编号集合
    - `startTime`、`endTime`：时间索引窗口（整数索引）
    - `downsample=auto|none|lttb|minmax`（默认 `auto`）
    - `targetPoints`：目标点数（默认 `5000`）
- `GET /api/datasets/:id/wavecanvas` - 获取 WaveCanvas 所需数据结构
- `GET/POST/DELETE /api/datasets/:id/annotations` - 管理标注（持久化到 `annotations.json`）

## Development Roadmap

See [docs/design.md](docs/design.md) for the complete design specification.

**M1 - Core (Current):**

- ✅ 上传与解析 COMTRADE（1991/1999/2013）
- ✅ 元数据提取（CFG）
- ✅ ECharts 交互式可视化（索引/时间双模式）
- ✅ `.dat` 实数据解析：`ascii`/`binary`/`binary32`/`float32`

**M2 - Performance:**

- ✅ 自动 LTTB 下采样（模拟量）与数字量状态抽取
- ⬜ Chunk 级索引与超大文件优化
- ⬜ SSE/WebSocket 流式加载

**M3 - Features:**

- ⬜ 标注读写（JSON 持久化）
- ⬜ 导出 PNG/CSV
- ⬜ 多数据集对比视图

**M4 - Production:**

- ✅ 登录鉴权（JWT，前端路由守卫）
- ⬜ Docker Compose 部署
- ✅ MinIO 存储后端（可选）

## Configuration

- 应用配置文件：`backend/config.yaml`（示例：`backend/config.yaml.example`）
- 环境变量覆盖：
  - 服务器端口：`SERVER_PORT`（默认 `8080`）
  - 存储类型：`STORAGE_TYPE=local|minio`
  - 本地存储路径：`STORAGE_LOCAL_PATH`（默认 `./data`）
  - MinIO：`MINIO_ENDPOINT`、`MINIO_ACCESS_KEY`、`MINIO_SECRET_KEY`、`MINIO_BUCKET`、`MINIO_USE_SSL`
  - 鉴权：`AUTH_USERNAME`、`AUTH_PASSWORD`、`AUTH_SECRET`

## Contributing

This is a demonstration project. For production use, consider:

- 更加健壮的 COMTRADE 解析（极端格式与异常容错）
- 完整的输入校验与错误处理
- 限流与安全加固
- 测试覆盖与 CI/CD

## License

MIT
