# Kanban Demo

独立的 Kanban 编排 Demo，服务端使用 Go + Gin，前端使用 React + Vite。

## 当前阶段

当前仅完成工程初始化和 HTTP 协议占位：

- `GET /api/health`：服务健康检查。
- `GET /events`：SSE 连接占位，后续接入事件总线。
- 如果存在 `web/dist`，Go 服务会托管构建后的前端静态文件。

领域状态机、内存 Store、Fake Harness 和完整 UI 将按 `AGENTS.md` 中的阶段顺序逐步实现。

## 环境要求

- Go 1.26+
- Node.js 20+
- npm 10+

## 开发启动

在一个终端启动 Go API：

```powershell
go run ./cmd/kanban-demo
```

在另一个终端启动 Vite：

```powershell
cd web
npm run dev
```

浏览器访问 `http://localhost:5173`。Vite 会将 `/api` 和 `/events` 代理到 Go 的 `http://localhost:8080`。

## 构建和检查

```powershell
go test ./...
cd web
npm run build
```

生产/演示模式先构建前端，再启动 Go 服务：

```powershell
cd web
npm run build
cd ..
go run ./cmd/kanban-demo
```

此时访问 `http://localhost:8080`。
