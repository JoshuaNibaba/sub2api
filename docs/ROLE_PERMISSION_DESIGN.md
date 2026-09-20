# 角色与权限设计

## 目标

当前版本把用户分成四种内置角色：

| 角色值 | 中文名称 | 定位 |
|---|---|---|
| `super_admin` | 超级管理员 | 继承历史 `admin` 的全部能力 |
| `admin` | 管理员 | 受限的后台管理能力 |
| `user` | 用户 | 现有普通用户能力 |
| `enterprise_user` | 大客户 | 用户能力 + 脱敏号池/用量/日志能力 |

数据库迁移 `239_expand_user_roles.sql` 会把已有 `admin` 记录迁移为 `super_admin`，保证升级后已有管理员不丢权限。新建的受限管理员使用 `admin`。

## 核心原则

角色不是授权判断的最终依据，权限才是。后端使用 `internal/domain/permissions.go` 中的权限标识和默认角色矩阵；路由中间件、业务服务和前端路由都应该依赖权限标识，不要新增散落的 `role == "admin"` 判断。

当前权限矩阵是代码内置的默认策略：

- `super_admin` 自动拥有所有已声明权限。
- `admin` 拥有面板、用户、分组、用量、运维读取和部分管理权限，但不拥有系统、凭据、支付配置、插件、审计等高风险权限。
- `user` 不拥有后台权限。
- `enterprise_user` 只拥有企业只读能力。

权限标识采用稳定的点号命名，例如：

```text
admin.users.read
admin.users.write
admin.users.role_manage
admin.credentials.read
admin.system.write
admin.ops.write
admin.usage.write
admin.audit.write
enterprise.account_pool.read
enterprise.usage.read
enterprise.logs.read
```

## 修改权限是否需要大改

不需要。默认矩阵集中在一个文件中。给角色增删已有权限时，只需修改：

```text
backend/internal/domain/permissions.go
```

然后运行后端测试即可。路由和页面只依赖权限标识，不需要修改数据库结构，也不需要批量修改用户数据。

新增一个权限时，通常只需要：

1. 在后端声明新的 `Permission` 常量。
2. 在默认角色矩阵中给需要的角色加入或移除它。
3. 在目标路由上增加 `RequirePermission` 或 `RequireReadWritePermission`。
4. 若前端需要隐藏菜单或按钮，在 `frontend/src/utils/permissions.ts` 增加同名常量和界面判断。
5. 添加权限矩阵测试和接口 401/403 测试。

这意味着“某个角色增加/删除一个已有权限”是小改动；“新增一个业务模块”才需要新增接口、DTO、界面和测试。

## 读写权限

对于同时包含查询和变更的路由组，使用：

```go
middleware.RequireReadWritePermission(
    service.PermissionAdminUsersRead,
    service.PermissionAdminUsersWrite,
)
```

GET/HEAD 请求检查 read 权限，POST/PUT/PATCH/DELETE 检查 write 权限。这样受限管理员可以读取允许范围的数据，但不能因为获得列表权限而获得写权限。

## 角色管理保护

创建或修改管理员角色需要 `admin.users.role_manage`，默认只授予超级管理员，并继续要求 step-up 二次验证。后端不信任前端下拉框；即使直接调用 API，也会拒绝受限管理员的角色提升、降级或超级管理员创建操作。

## 大客户数据边界

大客户能力不应复用 `/api/v1/admin/*`。后续专用接口应放在独立的 `/api/v1/enterprise/*` 路由下，并返回脱敏 DTO：

- 号池只返回账号 ID、显示名称、平台、状态、分组、模型摘要和健康/容量信息。
- 禁止返回 OAuth token、API Key、Cookie、代理认证、完整凭据 JSON 和完整上游 URL。
- 日志默认只允许本用户、授权分组或授权业务范围，并清理 Prompt、请求头、Token 和上游响应体。

## 企业只读接口

当前已提供三类专用接口，均位于 `/api/v1/enterprise/*`，不会复用管理员 URL：

- `GET /enterprise/account-pool`：分页返回脱敏号池状态、平台和容量信息。
- `GET /enterprise/usage-logs`：仅返回当前登录用户自己的用量记录。
- `GET /enterprise/error-logs`：仅返回当前登录用户自己的脱敏错误记录。

这些接口的后端权限分别是 `enterprise.account_pool.read`、`enterprise.usage.read` 和
`enterprise.logs.read`。前端菜单和路由只负责体验，后端权限检查和数据范围检查始终有效。

## 将来做成后台可配置

当前矩阵是代码内置的，优点是简单、可审查、部署后行为确定。以后如果需要在管理面板中动态调整角色权限，可以在不改变调用方的前提下增加：

1. `role_permissions` 表（role、permission、enabled、updated_at）。
2. 权限仓储和 Redis/进程缓存。
3. `PermissionResolver`：先读数据库覆盖，再回退到代码默认矩阵。
4. 超级管理员专用的角色权限编辑页和审计日志。

路由和服务层仍然调用同一个 `HasPermission`/`RequirePermission` 接口，因此不需要重写业务模块。这个演进路径可以把“改代码发布”升级为“超级管理员在面板中调整权限”，同时保留代码默认值作为灾备回退。

## 安全要求

- 前端权限只用于菜单、页面和按钮显示，不能代替后端授权。
- 权限变更、角色变更、敏感数据读取和高风险写操作必须进入审计日志。
- 新权限默认不开放给任何低权限角色，先写测试再加入角色矩阵。
- 任何涉及凭据、备份、系统更新、插件、支付密钥和原始审计日志的权限都应默认只给超级管理员。
