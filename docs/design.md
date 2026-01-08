# ComTrade Web 波形查看器设计文档 (Vue3 + Go)

## 1. 目标与范围
- **目标**: 通过浏览器交互式查看 ComTrade 文件（.cfg + .dat）中的波形与元数据，支持大文件的高效浏览、缩放、通道选择与基本标注。
- **非目标**: 不做复杂保护/控制算法分析、不实现完整的电力扰动分析套件；不强依赖特定厂商格式扩展。
- **用户画像**: 继保工程师、测试工程师、实验室与教学场景中需要快速查看与对比 ComTrade 波形的用户。

## 2. 背景与术语
- **ComTrade**: 电力系统暂态记录标准格式，通常包含 `.cfg`（配置信息，如通道、采样率、时间基准、比例系数等）与 `.dat`（采样数据）。
- **数据规模**: 典型文件从数 MB 到数百 MB；可能包含多路模拟/开关量通道。

## 3. 用户场景与用例
- **上传与预览**: 用户上传一组 `.cfg` + `.dat`，系统解析并呈现通道列表与波形预览。
- **选择与缩放**: 选择通道、缩放时间轴、查看细节（游标/十字线/统计）。
- **标注与导出**: 添加简单标注（区间/事件点），导出图或标注。
- **对比**: 加载多个数据集，进行通道对比或叠加（后续增强）。

## 4. 总体架构
- **前端 (Vue3 + Vite)**
  - 组件化波形查看器，支持高性能渲染与交互（缩放、拖拽、通道切换）。
  - 使用 `Pinia` 做全局状态管理，`Vue Router` 做页面路由。
  - 数据获取通过 REST（元数据与区块数据）+ WebSocket/SSE（进度/流式预览）。
  - 图表渲染库：ECharts
- **后端 (Go)**
  - Web 服务框架：Gin
  - 模块：文件上传与管理、ComTrade 解析、数据分块与降采样、流式传输、会话与缓存管理。
  - 存储：本地文件系统（数据集目录）；可扩展对象存储（S3）或持久化 DB（元数据索引）。
- **数据流**
  1) 前端上传 `.cfg` + `.dat` → 后端校验与持久化 → 解析 `.cfg` 生成元数据。
  2) 前端拉取元数据与预览（降采样）→ 用户交互选择时间窗与通道。
  3) 后端按需分块/降采样返回数据片段（JSON/二进制），前端增量绘制。

## 5. 前端设计
- **页面结构**
  - `UploadPane`: 拖拽/选择上传 `.cfg` + `.dat`，显示解析进度与错误信息。
  - `DatasetList`: 已上传数据集列表，支持删除/重命名。
  - `WaveformViewer`: 主查看器，包含时间轴缩放、通道叠加、游标与图例。
  - `ChannelSidebar`: 通道选择与分组（模拟量/开关量），搜索与可见性切换。
  - `TimeNavigator`: 小视窗预览与快速定位，支持窗口拖拽缩放。
  - `AnnotationPanel`: 简易标注（点/区间），导出标注。
  - `SettingsModal`: 绘制参数（色板、线宽）、降采样策略、单位显示。
- **状态管理 (Pinia)**
  - `datasetStore`: 当前数据集、通道列表、采样信息、加载进度。
  - `viewStore`: 选中通道、当前时间窗、缩放级别、图表配置。
  - `annotationStore`: 标注集合、筛选与导出。
- **渲染策略**
  - 大数据采用预览降采样 + 按需细节加载（窗口内高分辨率数据）。
  - 虚拟化绘制（仅绘制视窗范围），避免 DOM/Canvas 过载。
  - 通道分层渲染（模拟量为折线，开关量为阶梯）。

## 6. 后端设计
- **模块划分**
  - `ingest`: 上传/校验/存储文件；生成数据集 ID 与目录结构。
  - `parser`: 解析 `.cfg`（站点、记录信息、通道、比例与单位、采样率、触发时间等）；支持 `.dat` 数据读取（ASCII/Binary，依据 `.cfg`）。
  - `indexer`: 为 `.dat` 建立块级索引（按样本范围与字节偏移），便于随机访问。
  - `downsampler`: 预览降采样（LTTB/MinMax），细节请求可关闭或减弱降采样。
  - `stream`: SSE/WebSocket 推送解析进度与预览片段；HTTP 分块传输。
  - `api`: REST 接口（数据集管理、元数据、数据片段、标注）。
- **存储结构（本地）**
  - `data/{datasetId}/cfg` → 原始 `.cfg`
  - `data/{datasetId}/dat` → 原始 `.dat`
  - `data/{datasetId}/meta.json` → 解析后的元数据快照
  - `data/{datasetId}/index.bin` → 数据块索引（可选）
  - `data/{datasetId}/annotations.json` → 标注

## 7. 接口规范（草案）
- **上传与数据集管理**
  - `POST /api/datasets/import`
    - FormData: `cfg` (file), `dat` (file), `name` (optional)
    - 200: `{ datasetId, name }`
  - `GET /api/datasets`
    - 200: `[{ datasetId, name, createdAt, sizeBytes }]`
  - `DELETE /api/datasets/{id}`
    - 200: `{ ok: true }`
- **元数据与通道**
  - `GET /api/datasets/{id}/metadata`
    - 200: `{ station, recording, sampling, channels: [{ id, name, type, unit, scale }], timebase, startTime, endTime }`
- **数据获取（按需/窗口）**
  - `GET /api/datasets/{id}/waveforms`
    - Query: `channels=ch1,ch2`, `start=ms`, `end=ms`, `downsample=auto|none|lttb|minmax`, `targetPoints=5000`
    - 200: `{ series: [{ channelId, t: [..], y: [..] }], window: { start, end }, sampling: { rate }, downsample: { method, points } }`
    - 备注：支持 `Content-Type: application/json` 或 `application/octet-stream`（Float32/Int16 二进制，配合额外 header 描述）。
- **预览/进度（可选 SSE/WS）**
  - `GET /api/datasets/{id}/preview/stream` (SSE)
    - 事件：`progress`, `preview`（降采样片段）。
  - `WS /api/datasets/{id}/stream`
    - 消息：`{"type":"preview","payload":...}` / `{"type":"progress","payload":...}`
- **标注**
  - `GET /api/datasets/{id}/annotations`
    - 200: `[{ id, type: "point|range", channelId?, t|{start,end}, note }]`
  - `POST /api/datasets/{id}/annotations`
    - Body: `{ type, channelId?, t|{start,end}, note }`
    - 200: `{ id }`
  - `DELETE /api/datasets/{id}/annotations/{annId}`
    - 200: `{ ok: true }`

## 8. ComTrade 解析要点
- **.cfg 关键字段**: 站点名、记录设备、模拟/开关量通道数与详情、比例/偏移、单位、采样率、开始/结束时间、数据格式（ASCII/BINARY）、时间基准与触发。
- **.dat 读取**: 基于 `.cfg` 的格式描述进行逐样本读取；为大文件建立块索引（样本区间→文件偏移）。
- **数据类型**: 模拟量常为整数加比例与偏移，需转换为物理量；开关量为 0/1 阶梯。

## 9. 性能与扩展策略
- **分块与随机访问**: 解析时扫描建立索引；请求窗口时仅读取必要字节范围。
- **降采样**: 采用 LTTB（Largest-Triangle-Three-Buckets）或 Min/Max 以减少绘制点数，同时保留形状特征。
  - LTTB 思想：将 $N$ 点分桶为 $B$ 桶，选择每桶能与相邻桶形成最大面积三角形的点；从而在降维同时保持轮廓。
- **并发与流水线**: Go 协程并发读取与解码，异步缓存热点窗口。
- **内存映射（可选）**: 对超大 `.dat` 可用 `mmap` 进行只读映射以提升随机访问效率（需权衡兼容性）。
- **缓存**: 预览层级缓存（多分辨率金字塔）；LRU 缓存窗口数据；CDN 缓存静态前端资源。

## 10. 安全与合规
- **上传限制**: 文件大小上限、类型白名单（必须 .cfg + .dat）、扫描危险内容（拒绝不期望的可执行或路径穿越）。
- **输入校验**: 字段/参数范围校验，时间窗合法性，通道存在性。
- **路径安全**: 所有文件操作限定到数据根目录，拒绝相对路径上跳。
- **CORS/鉴权**: 允许前端域名；可选 JWT/Token；速率限制与 IP 限制。
- **隐私**: 最小化保留上传内容；支持按数据集清理与匿名化。

## 11. 测试与验收
- **单元测试**: `.cfg` 解析、比例/单位转换、`.dat` 解码与索引构建、降采样算法。
- **集成测试**: 端到端上传→解析→窗口拉取→渲染；大文件下的响应延迟与内存占用。
- **性能基准**: 针对不同文件大小与通道数，记录窗口请求的 P95/P99 延迟；绘制帧率与交互流畅度。
- **兼容性**: 浏览器（Chrome/Firefox/Edge）；Linux 服务器运行环境。

## 12. 部署与运维
- **容器化**: 前端构建为静态资源；后端 Go 构建为单独服务镜像；通过 Docker Compose 编排。
- **反向代理**: Nginx/Traefik 提供静态资源与 API 代理、TLS、压缩。
- **日志与监控**: 结构化日志、请求性能指标（Prometheus + Grafana）、错误告警。
- **存储管理**: 数据集生命周期与清理策略（LRU/TTL），磁盘配额与健康检查。

## 13. 里程碑
- **M1 核心功能**: 上传→解析 `.cfg` → 基本预览与窗口数据拉取（模拟量至少），ECharts 渲染与缩放；基础测试。
- **M2 性能与流式**: 建索引、降采样与并发优化；SSE/WS 预览与进度；缓存策略。
- **M3 完善**: 标注、导出、开关量显示优化、通道分组与搜索；Docker 部署与监控。
- **M4 扩展**: 多数据集对比、对象存储支持、鉴权与多用户会话。

## 14. 实施建议与目录结构
```
comTradeViewer/
  frontend/            # Vue3 + Vite 项目
  backend/             # Go 服务
  docs/
    design.md          # 本设计文档
  data/                # 本地数据集存储（运行时生成）
```

### 前端初始化（建议）
```
npm create vite@latest frontend -- --template vue
cd frontend
npm i echarts pinia vue-router axios
npm run dev
```

### 后端初始化（建议）
```
mkdir -p backend
cd backend
go mod init comtradeviewer
go get github.com/gin-gonic/gin
```

## 15. 附录：数据结构示例
- **元数据响应示例**
```json
{
  "station": "Station A",
  "recording": { "device": "REC-01" },
  "sampling": { "rate": 19200 },
  "channels": [
    { "id": "A1", "name": "Ia", "type": "analog", "unit": "A", "scale": { "k": 0.001, "b": 0 } },
    { "id": "D1", "name": "Trip", "type": "digital", "unit": null }
  ],
  "timebase": 1e-6,
  "startTime": 1736000000000,
  "endTime": 1736000100000
}
```
- **波形窗口响应示例（JSON）**
```json
{
  "series": [
    { "channelId": "A1", "t": [0, 0.052, 0.104], "y": [0.1, 0.2, 0.15] },
    { "channelId": "D1", "t": [0, 0.052, 0.104], "y": [0, 0, 1] }
  ],
  "window": { "start": 0, "end": 0.5 },
  "downsample": { "method": "lttb", "points": 5000 }
}
```

---

### 设计抉择与权衡
- 选用 Vue3 + ECharts 获得较好的交互与性能；复杂自定义可退回 D3。
- Go + Gin 提供简单高效的 API；如需更强性能可探索二进制协议与 `mmap`。
- 先实现可用的降采样与分块访问，再完善高级功能（对比/注释导出）。
