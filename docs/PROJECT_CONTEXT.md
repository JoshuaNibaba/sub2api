# Sub2API 项目上下文与开发基线

> 这份文档是对当前 Fork 的代码勘察结果，供后续功能修改、测试和发布使用。
> 它描述代码现状，不替代 `README_CN.md`、`DEV_GUIDE.md` 或部署文档。

## 1. 当前仓库状态

- 本地项目：`/Users/joshua/Source/sub2`
- 个人仓库：`https://github.com/JoshuaNibaba/sub2api.git`
- 官方上游：`https://github.com/Wei-Shaw/sub2api.git`
- 当前基线：`main` 与两个远程的 `main` 均指向提交
  `bbdcfbac0a3c0982a33639fba372d55d25724e2f`
- Git 远程约定：
  - `origin`：个人 Fork，功能分支和个人版本推送到这里。
  - `upstream`：官方仓库，只用于同步和对比，不直接作为个人发布目标。

后续功能默认在独立分支上开发，验证通过后再合并或推送到个人仓库的目标分支。除非另有说明，不直接在 `main` 上堆积未验证的改动。

## 2. 产品与运行时边界

Sub2API 是一个 AI API 网关和订阅配额管理平台，主要职责包括：

1. 接收下游客户端的 API 请求并完成 API Key 鉴权、分组准入、模型路由、账号调度、失败切换和计费。
2. 将请求转发到 Claude、OpenAI、Gemini、Grok 以及若干 OpenAI 兼容平台。
3. 提供用户面板和管理员面板，用于用户、分组、账号、渠道、代理、订阅、支付、用量和运维管理。
4. 通过 PostgreSQL 持久化业务数据，通过 Redis 承担缓存、队列、并发槽位、会话和短期状态。
5. 构建时可以将 Vue 前端嵌入 Go 二进制，由同一个 HTTP 服务提供页面和 API。

项目包含多种上游协议适配和账号 OAuth/Token 刷新逻辑。涉及凭据、代理、计费、调度、内容审核和安全审计的修改，应优先理解现有服务接口与测试，不要在路由层直接复制业务逻辑。

## 3. 目录地图

```text
.
├── backend/
│   ├── cmd/server/              # 主程序入口、版本信息、Wire 注入入口
│   ├── ent/schema/              # Ent 数据模型定义（修改后需重新生成）
│   ├── ent/                     # Ent 生成代码，不手工编辑
│   ├── internal/config/         # YAML/环境变量配置、默认值和校验
│   ├── internal/server/         # HTTP Server、全局中间件和路由注册
│   │   ├── routes/              # 面板、支付和网关路由分组
│   │   └── middleware/          # 鉴权、限流、审计、CORS、CSP、请求限制等
│   ├── internal/handler/        # HTTP 入参解析、响应和协议层编排
│   ├── internal/service/        # 领域业务、网关转发、调度、计费和后台任务
│   ├── internal/repository/     # Ent/SQL/Redis 访问和外部客户端实现
│   ├── internal/model/          # 持久化或传输模型辅助类型
│   ├── internal/pkg/            # 协议、HTTP、OAuth、日志等通用包
│   ├── migrations/              # PostgreSQL 增量迁移
│   └── internal/web/            # 前端构建产物的 embed/静态服务
├── frontend/
│   ├── src/api/                 # 面板 API 客户端
│   ├── src/components/          # 可复用 Vue 组件
│   ├── src/views/               # 页面级组件
│   ├── src/router/              # 页面路由和导航守卫
│   ├── src/stores/              # Pinia 状态
│   ├── src/composables/         # 组合式逻辑
│   ├── src/i18n/                # 国际化资源和检查
│   └── vite.config.ts           # 开发代理和生产构建输出
├── deploy/                     # Docker Compose、容器、安装和部署文档
├── docs/                       # 功能/接口/开发说明
├── openspec/                   # 变更提案和规格资料
└── .github/workflows/          # CI、发布和安全扫描
```

## 4. 后端启动与依赖注入

### 4.1 启动路径

入口是 `backend/cmd/server/main.go`：

1. 初始化引导日志并解析 `-setup`、`-version` 参数。
2. 首次运行时检查配置/数据库是否完成初始化。
3. `AUTO_SETUP` 开启时从环境变量自动初始化；否则启动设置向导服务。
4. 正常运行时加载配置，调用 Wire 生成的 `initializeApplication` 构建应用。
5. 启动后台服务和 HTTP Server，收到 `SIGINT`/`SIGTERM` 后执行有超时的优雅关闭。

依赖注入的声明入口是 `backend/cmd/server/wire.go`，生成文件是同目录的 `wire_gen.go`。ProviderSet 将配置、repository、service、middleware、handler、支付和 server 层连接起来。新增构造函数或依赖后，应更新 ProviderSet 并重新生成 Wire 代码，不要只修改生成文件。

### 4.2 配置

配置代码集中在 `backend/internal/config`：

- 支持 `config.yaml` 和环境变量；环境变量使用点号到下划线的映射。
- `LoadForBootstrap` 允许启动阶段暂缺 JWT secret；完整运行配置由 `Load` 校验。
- 配置覆盖服务器、数据库、Redis、网关超时/调度、代理、OAuth、支付、对象存储、审计和安全策略。
- 生产部署优先使用 `deploy/config.example.yaml`、`deploy/.env.example` 和部署文档，不把凭据提交到仓库。

修改配置字段时，同时检查默认值、环境变量可达性测试和部署示例；需要动态生效的设置还要检查 `SettingService` 的刷新回调。

## 5. HTTP 路由与请求链路

### 5.1 全局中间件

`backend/internal/server/router.go` 负责安装全局中间件，当前顺序包括：

1. 请求访问日志和客户端会话绑定上下文。
2. 普通日志、CORS、安全响应头/CSP、可选的 Server-Timing。
3. 嵌入式前端服务（跳过 API 路由），并在设置变化时刷新 HTML/`frame-src` 缓存。
4. `registerRoutes` 注册健康检查、面板 API、网关和页面接口。

修改中间件时要关注顺序：鉴权、客户端 IP、审计、限流和请求体限制都依赖上下文是否已准备好。

### 5.2 路由族

- `GET /health`：健康检查。
- `GET /setup/status`：前端判断是否需要设置向导。
- `/api/v1/auth`：注册、登录、2FA、OAuth、会话等认证接口。
- `/api/v1/user`、`/api/v1/keys`、`/api/v1/usage`、`/api/v1/subscriptions`：当前用户面板接口。
- `/api/v1/admin/*`：管理员面板，统一使用管理员鉴权；变更和敏感读取接入审计，部分操作需要 step-up/TOTP。
- `/api/v1/payment/*`：用户订单、公开恢复、支付回调和管理员支付管理。
- `/api/v1/model-plaza`：模型广场，可匿名或携带可选 JWT 访问，受设置和后端模式控制。
- `/v1/*`：主要 API 网关入口，包括 Claude Messages、OpenAI Responses/Chat Completions、模型、Embedding、图片、视频、语音和 WebSocket 等。
- `/v1beta/*`：Gemini 兼容入口。
- `/backend-api/codex/*`：Codex 相关直连/兼容入口。
- `/antigravity/v1*`：Antigravity 兼容入口。

### 5.3 网关请求链路

`backend/internal/server/routes/gateway.go` 先做路由级组合，再进入 handler/service：

1. 请求体大小、请求 ID、运维错误记录和入站端点规范化。
2. API Key 鉴权与订阅/分组检查。
3. 分组模型白名单校验。
4. Composite 路由解析，确定实际平台和上游模型。
5. 按平台选择 Claude、OpenAI、Gemini、Grok 或兼容实现。
6. Handler 解析请求、执行安全审核/计费前置检查，再调用 service 选择账号并转发。
7. Service 负责 Token 获取、代理/TLS 指纹、调度、失败切换、流式响应、用量记账和错误记录。

同一路径可能根据分组平台进入不同 handler。例如 `/v1/messages` 和 `/v1/responses` 会在路由层判断平台后分别走 OpenAI 兼容实现或通用网关实现。修改网关功能时要同时检查：路由条件、请求模型解析、上游请求体重写、账号调度、计费和失败切换。

## 6. 后端分层约定

### Handler 层

位置：`backend/internal/handler`。

负责 Gin 上下文、鉴权结果、请求体读取/校验、响应格式和协议边界。复杂业务应下沉到 service。网关 handler 文件按协议或能力拆分，例如 OpenAI Responses、图片、视频、Gemini、WebSocket 等。

### Service 层

位置：`backend/internal/service`。

负责领域逻辑和外部调用编排，包括账号/分组/API Key、网关路由、账号调度、OAuth、计费、用量、订阅、支付、运维、渠道监控、批量图片和插件等。Service 通过接口依赖 repository 或外部客户端，便于单元测试替换 stub。

### Repository 层

位置：`backend/internal/repository`。

负责 Ent、原生 SQL、Redis 缓存/队列以及 OAuth/上游 HTTP 客户端实现。复杂查询可以使用底层 `*sql.DB`，但应保持事务边界清晰，并为关键行为补单元或集成测试。

### 数据模型与迁移

- `backend/ent/schema` 是 Ent schema 的源代码。
- `backend/ent` 是生成代码，禁止手工修改。
- `backend/migrations` 是按序执行的 PostgreSQL 增量迁移；同一业务变更需要同时考虑 schema、迁移、repository 查询和回滚/兼容窗口。
- 修改 schema 后至少运行 `go generate ./ent`，若修改依赖注入再运行 `go generate ./cmd/server`，并提交生成结果。

## 7. 前端架构

前端使用 Vue 3、TypeScript、Pinia、Vue Router、Vue I18n、Vite 和 Vitest：

- `frontend/src/main.ts` 初始化主题、Pinia、注入配置、国际化、路由和应用挂载。
- `frontend/src/router/index.ts` 按公开页、用户页、管理员页和设置向导组织懒加载路由，并执行认证/管理员/功能开关守卫。
- `frontend/src/api` 封装后端请求；新增接口先补 API 模块和类型，再接入 store/view。
- `frontend/src/stores` 保存认证、设置、公告、支付、订阅等跨页面状态。
- `frontend/src/views` 是页面入口，`components` 负责可复用 UI，`composables` 负责可组合的交互/数据逻辑。
- `frontend/vite.config.ts` 将开发请求代理到 `VITE_DEV_PROXY_TARGET`（默认 `http://localhost:8080`）。
- 生产构建输出到 `backend/internal/web/dist`；后端使用 `-tags embed` 时把该目录嵌入二进制。

修改界面时要同步检查中英文/日文资源键、路由守卫、权限状态、错误处理和窄屏布局。`pnpm-lock.yaml` 必须与 `package.json` 一起维护。

## 8. 本地验证命令

在仓库根目录执行：

```bash
# 前后端总构建
make build

# 后端单元测试、集成测试和 lint
cd backend
go test -tags=unit ./...
go test -tags=integration ./...
golangci-lint run ./...

# 前端依赖、检查和构建
cd ../frontend
pnpm install --frozen-lockfile
pnpm run lint:check
pnpm run typecheck
pnpm run test:run
pnpm run build
```

根目录 `make test` 还会运行前端关键 Vitest 集合；CI 额外运行部署脚本检查和安全扫描。当前仓库期望 Go 1.27、Node.js 18+、pnpm 9、PostgreSQL 15+、Redis 7+，实际执行前以 `backend/go.mod`、锁文件和 CI 配置为准。

开发时建议先运行与改动直接相关的最小测试，再运行对应层级的完整测试。集成测试通常需要 PostgreSQL/Redis；没有依赖服务时不要把环境缺失误判成代码回归。

## 9. 功能修改的推荐路径

### 只改后端 API 或网关行为

1. 先定位 `routes` 的入口和 middleware 条件。
2. 在 handler 中确认请求/响应协议边界。
3. 在 service 中实现业务、调度、计费或转发变化。
4. 如需持久化，更新 repository、Ent schema 和 migration。
5. 添加或更新 handler/service/route 测试，并核对日志、审计和错误分类。

### 只改管理面板功能

1. 后端：`routes/admin.go` → `handler/admin` → `service` → `repository`。
2. 前端：`src/api` → `src/stores`（如有跨页状态）→ `src/views`/`src/components`。
3. 检查管理员权限、合规确认、step-up、审计日志和分页/筛选行为。

### 增加数据库字段或业务实体

1. 先修改 `backend/ent/schema`。
2. 创建新的递增 migration，并考虑旧版本滚动部署兼容。
3. 运行 Ent 生成，检查 repository/service 读写路径。
4. 补充迁移测试、repository 集成测试和 API 行为测试。

### 增加第三方平台或上游适配

统一通过 service 接口、HTTP client、Token/OAuth provider 和配置注入接入。不要在 handler 中直接保存上游凭据或拼接未经校验的 URL；同时检查代理、TLS 指纹、超时、错误透传、用量记账和隐私/安全审计。

## 10. Git 与上游同步

```bash
# 开始新功能
git switch main
git fetch upstream
git merge --ff-only upstream/main
git push origin main
git switch -c feat/<short-name>

# 修改、测试、提交并推送到个人仓库
git add <files>
git commit -m "feat: <summary>"
git push -u origin feat/<short-name>
```

在同步上游前先确认工作区干净，并检查是否存在个人分支未合并的改动。不要用 `git reset --hard` 或强制推送覆盖用户代码，除非用户明确要求。

## 11. 当前开发注意事项

- `DEV_GUIDE.md` 包含历史环境和其他 Fork 的信息；本项目以本文件开头记录的 `JoshuaNibaba/sub2api` 和当前机器环境为准，不复制其中的凭据或过时账号信息。
- API Key、OAuth token、数据库密码、JWT/TOTP secret、代理认证信息和完整上游 URL 不写入提交、日志或文档。
- 网关错误不能只看最终 HTTP 状态；需要同时看请求入口、上游尝试、账号切换、代理归因、计费和审计事件。
- 生成代码、锁文件、迁移文件和嵌入式前端产物都可能是构建输入；变更后检查 Git diff 是否包含应提交的生成结果。
- 功能开关通常同时存在于配置、系统设置、后端 handler/service 和前端 store/view，不能只修改其中一层。
