# MCPZERO.IO 项目设计书 (Project Design Document)

## 一、 项目愿景 (Vision)
**MCPZERO.IO** 旨在成为大模型时代最轻量、最高效的 **MCP (Model Context Protocol) 路由网关与变现基础设施**。平台致力于帮助开发者实现 **“Zero-Config 部署、Zero-Leak 安全、Zero-Friction 变现”**。
无论开发者的 MCP Server 运行在本地（通过 MCPZERO Tunnel）还是托管在云端（通过 MCPZERO Cloud），平台都为其提供统一的 **Streamable HTTP/SSE 远程端点、金融级 API Key 鉴权保护、全链路调用可视化看板以及按次扣费的分账结算层**。

---

## 二、 核心功能模块 (Core Features)

1. **MCPZERO Tunnel (穿透层)：** 开发者通过本地轻量级客户端（`mcpzero` CLI），一行命令将本地 Stdio 模式的 MCP Server 转换为公网可访问的远程端点，无需配置域名和 SSL 证书。
2. **Edge Auth Gateway (安全网关层)：** 部署在 Cloudflare 全球边缘节点，拦截所有客户端请求，进行毫秒级的 API Key 校验和多租户余额检查（Rate Limiting & Metering）。
3. **Observability Ledger (可视化账本)：** 异步解析 JSON-RPC 请求与返回体，全链路追踪 AI Agent 的每一次工具调用（Tool Call），提供精美的 Web 端链路回放看板。
4. **Monetization Engine (变现引擎)：** 支持 Web2 友好的信用卡充值与积分余额扣费系统，支持按 Tool 细粒度定价，自动完成平台抽成与开发者分账。

---

## 三、 技术栈选型 (Tech Stack)

| 组件 | 选用技术 | 选型理由 |
| :--- | :--- | :--- |
| **Edge Compute (网关/API)** | **Cloudflare Workers** | 全球边缘部署，冷启动时间接近 0，完美支持流式传输 (SSE/Stream) |
| **Database (关系型数据)** | **Cloudflare D1** | 原生 Serverless SQL 数据库，无网络 I/O 损耗，完美存储用户、API Key、计费规则、对账单 |
| **Session & Cache (缓存/鉴权)** | **Cloudflare KV / Durable Objects** | 极速读取 API Key 状态与余额，控制网关层校验延迟在 5ms 以内 |
| **Tunnel Control (隧道控制)** | **WebSocket (via Workers)** | 用于维持本地客户端与云端网关之间的长连接双向通信 |
| **Frontend Dashboard (控制面板)** | **Remix / Next.js on CF Pages** | 配合 TailwindCSS 构建极致流畅的可视化看板 |
| **User Auth (用户认证)** | **Clerk / Lucia Auth + D1** | 替代 Supabase Auth，保持 100% 纯粹的 Cloudflare 边缘生态 |

---

## 四、 系统架构设计 (System Architecture)

### 1. 流量与数据链路拓扑

```
[ AI Client (Cursor / Claude) ] 
         │ 
         │ (1) 带有 X-MCP0-Token 的 HTTP POST/SSE 请求
         ▼
 ┌────────────────────────────────────────────────────────┐
 │ Cloudflare Workers (MCPZERO Edge Gateway)              │
 │                                                        │
 │  ├── (2) 校验 Key & 余额 ──> [ CF KV / Durable Objects ]│
 │  └── (3) 异步写入调用日志 ─> [ CF D1 Database ]        │
 └────────────────────────────────────────────────────────┘
         │
         │ (4) 通过已建立的 WebSocket 隧道转发加密后的 JSON-RPC
         ▼
 [ 用户本地 / 托管容器 (mcpzero CLI) ]
         │
         │ (5) 管道桥接 (Pipes)
         ▼
 [ 开发者原生的 MCP Server (Stdio) ]
```

### 2. 核心交互流程

1. **建立连接：** 开发者在本地运行 `mcpzero --token xyz`，与 Cloudflare Workers 建立一条持久的 WebSocket 隧道，Workers 在 KV 中标记该用户隧道在线。
2. **请求鉴权：** AI Client 访问 `https://gw.mcpzero.io/v1/user_abc`，网关从 KV 中秒级验证 Header 中的 API Key 状态及买家点数余额。
3. **隧道通信与转换：** 鉴权通过后，网关将 HTTP 请求体包裹为 JSON-RPC，通过 WebSocket 灌入本地 CLI。CLI 将其写入本地 MCP 的 `stdin`。
4. **流式返回与记账：** 本地 MCP 的 `stdout` 吐出结果，CLI 通过 WebSocket 回传给网关。网关一方面流式（SSE）响应给 AI Client，另一方面触发 D1 数据库记录本次 Tool 扣费及审计日志。

---

## 五、 D1 数据库设计 (Database Schema - 核心表)

```sql
-- 用户表 (Users)
CREATE TABLE users (
    id TEXT PRIMARY KEY,
    email TEXT UNIQUE NOT NULL,
    role TEXT DEFAULT 'developer', -- developer / buyer
    balance INTEGER DEFAULT 0,     -- 用户积分余额 (单位: 美分/点)
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- MCP 服务端点表 (Endpoints - 兼容隧道与未来托管)
CREATE TABLE mcp_endpoints (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,           -- 'tunnel' / 'cloud_hosting'
    status TEXT DEFAULT 'offline', -- 'online' / 'offline'
    pricing_model TEXT,            -- JSON 字符串，例如 {"read_db": 10, "list_tools": 0}
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- API Keys 表 (给买家/客户端调用)
CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    key_hint TEXT NOT NULL,        -- 密钥脱敏显示 (如: mz_live_...xxxx)
    encrypted_key_hash TEXT NOT NULL,
    status TEXT DEFAULT 'active',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);

-- 审计与计费对账表 (Ledger / Logs)
CREATE TABLE usage_ledger (
    id TEXT PRIMARY KEY,
    endpoint_id TEXT NOT NULL,
    buyer_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    status TEXT NOT NULL,          -- 'success' / 'error' / 'timeout'
    cost INTEGER DEFAULT 0,        -- 本次调用扣除点数
    request_payload TEXT,          -- 异步存储(可选，可脱敏)
    response_payload TEXT,         -- 异步存储(可选，可脱敏)
    latency_ms INTEGER,            -- 耗时
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY(user_id) REFERENCES users(id)
);
```

---

## 六、 项目路线图 (Milestones & Roadmap)

### Phase 1: MVP 验证阶段 (1-2 周)
* [ ] 编写基于 Node.js/Go 的轻量级本地 CLI 原型（`mcpzero`）。
* [ ] 实现 Cloudflare Workers WebSocket 隧道服务，完成 `HTTP SSE <-> WebSocket <-> Stdio` 的双向流式协议转换。
* [ ] 在 Workers 中加入固定 API Key 的硬编码鉴权，跑通 Cursor 远程连接本地 MCP。

### Phase 2: 控制面板与全链路监控 (2-3 周)
* [ ] 引入 Clerk 或 Lucia Auth 实现 Cloudflare Pages 前端管理面板登录。
* [ ] 设计 D1 数据库结构，实现多租户 API Key 的生成、吊销管理。
* [ ] 开发**可观测性看板（Observability Dashboard）**：Workers 在转发流量时异步写日志到 D1，前端渲染出精美的 Tool 调用链路图。

### Phase 3: 计费引擎与开发者商店 (3-4 周)
* [ ] 在网关层引入 基于 Durable Objects 或 KV 的高并发内存账户扣费扣点逻辑。
* [ ] 对接 Stripe 支付，实现买家法币充值 Credits，卖家绑定账户提现。
* [ ] 上线 **MCPZERO Marketplace** 首页，允许开发者一键将自己的本地隧道公开并标记价格上架。

### Phase 4: 升级云端常驻托管 (未来规划)
* [ ] 引入边缘容器或与第三方轻量级容器平台（如 Kapsule/Fly.io）API 对接，允许用户将镜像一键从本地代码升级为 **MCPZERO Cloud 常驻托管**。
```