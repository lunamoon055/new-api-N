# GHLINK API 项目整理与改动记录

更新时间：2026-06-09  
代码分支：`main`  
最新提交：`85e8ef1 添加创作中心模型描述管理`

## 1. 项目概览

GHLINK API 基于 `new-api` 项目改造，是一个 AI API 网关与管理后台。它将 OpenAI、Claude、Gemini、Sora、Kling 等不同上游能力统一到同一套 API、渠道、模型、计费、用户和管理后台中。

当前项目主要由两部分组成：

- 后端：Go + Gin + GORM，负责用户、渠道、模型、计费、API 转发、异步任务、系统配置等。
- 前端：React + TypeScript + Rsbuild + Tailwind，位于 `web/default/`，负责首页、控制台、创作中心、模型广场、系统设置等页面。

重要约束：

- 保留原项目的版权、署名、项目身份信息，不删除 `new-api` / `QuantumNous` 相关声明。
- 后端 JSON 编解码遵循 `common/json.go` 封装。
- 数据库需兼容 SQLite、MySQL、PostgreSQL。
- 前端优先使用 Bun。

## 2. 当前部署状态

### 2.1 旧服务器

旧服务器信息来自前期部署：

- SSH：`root@p58an0rAAhg7`
- 端口：`22`
- 访问 IP：`154.21.198.132`

2026-06-09 重新探测结果：

- `http://154.21.198.132/` 返回 `HTTP/1.1 200 OK`
- `http://154.21.198.132/creation` 返回 `HTTP/1.1 200 OK`
- 响应头中存在 `Via: 1.1 Caddy`，说明旧服务器仍有 Web 服务响应。

### 2.2 Zeabur

已将代码推送到 GitHub 仓库 `lunamoon055/new-api-N` 的 `main` 分支，Zeabur 服务应从该仓库拉取部署。

2026-06-09 重新探测结果：

- `https://ghlink.zeabur.app/` 返回 `HTTP/2 404`
- `https://ghlink.zeabur.app/creation` 返回 `HTTP/2 404`

这说明当前 Zeabur 域名没有接到应用服务，或者域名/网关仍处于未绑定、未 provision、服务未运行等状态。代码已经推到远端，但 Zeabur 域名当前不能作为可访问上线状态。

## 3. 提交时间线

| 提交 | 内容 |
| --- | --- |
| `51a2a2c first commit` | 导入完整项目基础代码，包括后端、前端、文档、Docker、i18n、管理后台等。 |
| `470fd2a 异步` | 初步接入创作中心异步媒体 API、视频任务路由、前端提交和历史逻辑。 |
| `74c0d5d Restore creation center async video fixes` | 恢复并修正异步视频任务、Sora 适配器、任务轮询、前端视频状态处理。 |
| `3a3764f Serve creation center from default frontend` | 让默认前端能正常服务 `/creation` 路由，修正嵌入静态文件和 Web 路由。 |
| `44cb7b0 主页` | 调整主页初版样式。 |
| `fa2f409 Integrate Grainient hero and simplify home page` | 接入 React Bits Grainient 背景，简化主页内容。 |
| `e5061b7 创作中心筛选` | 增加创作中心聊天、图片、视频模型筛选和超级管理员模型分类配置。 |
| `5ead4c9 完善创作中心模型分类和素材发送` | 完善模型分类、素材上传、发送逻辑、消耗展示、Enter 发送。 |
| `7b74093 修正创作中心零消耗展示` | 修复零消耗模型价格字段被省略的问题。 |
| `85e8ef1 添加创作中心模型描述管理` | 增加超级管理员模型描述管理入口和后端配置项。 |

## 4. 首页改动

主要文件：

- `web/default/src/features/home/components/sections/hero.tsx`
- `web/default/src/features/home/components/grainient.tsx`
- `web/default/src/features/home/components/grainient.css`
- `web/default/src/features/home/index.tsx`
- `web/default/package.json`

已完成调整：

- 使用 React Bits 的 `Grainient` 作为主页第一屏背景。
- 新增 `ogl` 依赖支持 WebGL 渐变背景。
- 首页文案调整为两行：
  - `欢迎使用GHLINK API`
  - `尽情发挥你的想象力`
- 文字改成白色，并减小到更适合第一屏展示的尺寸。
- 移除了主页下面的功能、流程、统计等大段营销内容，只保留第一屏视觉。
- 保留公共顶部导航和状态栏逻辑，没有改动用户要求保留的导航状态栏代码。
- 自定义首页内容仍然兼容：如果后台配置了自定义首页内容或 URL，仍会优先显示后台配置内容。

## 5. 创作中心页面

主要文件：

- `web/default/src/routes/creation/index.tsx`
- `web/default/src/features/creation-center/index.tsx`
- `web/default/src/features/creation-center/api.ts`
- `web/default/src/features/creation-center/session.ts`
- `web/default/src/features/creation-center/types.ts`

### 5.1 页面结构

新增 `/creation` 页面，挂载 `CreationCenter` 组件。

当前创作中心包含：

- 左侧模型与模式栏
  - `聊天`
  - `图片`
  - `视频`
- 可用模型列表
- 当前模型预览区域
- 创作工作区
  - 预览
  - 素材
  - 历史
- 底部输入栏
  - 素材添加
  - 提示词输入
  - 发送按钮
  - 视频清晰度和时长选择

### 5.2 模型筛选

创作中心模型按用途分为三类：

- `chat`：聊天、写作、代码、分析类模型。
- `image`：图片生成模型。
- `video`：视频生成模型。

筛选来源：

1. 优先读取超级管理员手动分类配置。
2. 没有手动配置时，根据模型支持的 endpoint 类型自动判断。
3. 对已知模型使用内置元数据兜底，例如 `gpt-image2`、`kling-v3`、`sora2`。

### 5.3 可用模型描述与标签

模型展示内容包括：

- 模型名称
- 模型描述
- 模型标签
- 模型消耗摘要
- endpoint 类型
- vendor 信息

描述优先级：

1. 超级管理员手动描述。
2. 定价/模型配置里的描述。
3. 代码内置默认描述。

标签中会过滤前端不需要展示的 `async` 标签，避免用户侧看到过于技术化的说明。

### 5.4 历史记录

创作历史保存在浏览器 `localStorage` 中。

实现细节：

- storage key：`creation-center-history:{userId 或 guest}`
- 最多保留 20 条。
- 每条历史包含：
  - 会话 ID
  - 创建时间
  - 模式
  - 模型
  - prompt
  - 素材摘要
  - 结果
  - 视频参数
- 刷新页面后仍可恢复本地浏览器里的历史记录。

限制：

- 历史目前是浏览器本地保存，不是服务端数据库会话。
- 换浏览器、清缓存、换设备后历史不会同步。

### 5.5 素材处理

素材目前是前端预览级能力。

已支持：

- 图片素材：小于等于 4MB 时转成 data URL，可传给支持图片输入的聊天请求。
- 文本素材：支持 `.csv`、`.json`、`.md`、`.txt`、`.tsv`、`.yaml`、`.yml` 等，最多读取前 8000 字符。
- 提交时会将素材摘要拼进 prompt，格式为 `参考素材 / Reference assets`。

限制：

- 素材还没有接入服务端文件库。
- 大文件、视频文件目前不会真正上传到后端。

## 6. 创作中心后端 API

主要文件：

- `controller/creation.go`
- `controller/creation_test.go`
- `controller/option.go`
- `dto/creation.go`
- `router/api-router.go`
- `docs/creation-center-api.md`

### 6.1 模型目录接口

接口：

```http
GET /api/creation/models
GET /api/creation/models?mode=chat
GET /api/creation/models?mode=image
GET /api/creation/models?mode=video
```

特点：

- 支持匿名浏览。
- 登录用户会根据可用分组过滤模型。
- 匿名用户只看到公开可用分组。
- 响应中不会暴露敏感定价字段、渠道 ID、billing expression、group 配置等。
- 返回模型分组、模型列表、vendor 摘要。

### 6.2 消耗展示

后端将模型价格转换成前端安全可展示的 `cost` 摘要。

支持三类：

- `per_token`
  - `input_price_per_million`
  - `output_price_per_million`
- `per_request`
  - `request_price`
  - `request_quota`
- `dynamic`
  - 动态表达式计费，不暴露表达式内容。

已修复的问题：

- 零价格模型原本会因为 `omitempty` 被省略字段。
- 现在改成指针字段，显式保留 `0`，前端能正常显示零消耗。

### 6.3 图片与视频提交

图片接口：

```http
POST /api/creation/images/generations
```

视频异步接口：

```http
POST /api/creation/video/async-generations
GET /api/creation/video/async-generations/:taskId
```

前端发送逻辑：

- `chat` 模式走 `/pg/chat/completions`
- `image` 模式走 `/api/creation/images/generations`
- `video` 模式走 `/api/creation/video/async-generations`

提交真实任务前需要登录。

### 6.4 异步视频任务

已做调整：

- 前端提交视频后先拿任务 ID。
- 结果区显示 queued / processing / completed / failed / unknown 状态。
- 支持手动刷新任务状态。
- 增加生成倒计时，按清晰度和时长估算生成时间。
- 支持从任务响应中解析 `url`、`result_url`、`output_url`、`video_url` 等字段。

视频参数：

- 清晰度：
  - `1080`
  - `2K`
  - `4K`
- 普通视频模型时长：
  - `5s`
  - `10s`
  - `15s`
- `sora2` 专属时长：
  - `4s`
  - `8s`
  - `12s`

### 6.5 异步视频渠道适配

相关文件：

- `relay/channel/task/sora/adaptor.go`
- `relay/channel/task/sora/adaptor_test.go`
- `controller/video_proxy.go`
- `service/task_polling.go`

已做调整：

- 增强 Sora 异步任务适配。
- 修正 task polling 的状态处理。
- 增加异步任务测试。
- 增强视频代理对上游不同返回结构的兼容。

## 7. 超级管理员管理入口

### 7.1 模型分类管理

配置 key：

```text
CreationModelCategories
```

用途：

- 允许超级管理员手动把模型归类到 `chat`、`image`、`video`。
- 解决模型 endpoint 信息不够准确时，前端筛选错误的问题。
- 例如可以把 `ko3` 手动归到图片，把 `video-2.0` 归到视频。

权限：

- 前端只有 `ROLE.SUPER_ADMIN` 显示入口。
- 后端保存走 `/api/option/`，该路由使用 `RootAuth()`。

校验：

- 必须是 JSON 对象。
- 模型名不能为空。
- 分类只能是 `chat`、`image`、`video`。

### 7.2 模型描述管理

配置 key：

```text
CreationModelDescriptions
```

用途：

- 允许超级管理员为所有可见创作模型手动填写描述。
- 手动描述优先级最高。
- 空字段不会保存，会继续使用自动描述。
- 弹窗中已手动填写过的描述会回填，便于继续编辑。

权限：

- 前端只有 `ROLE.SUPER_ADMIN` 显示“管理描述”入口。
- 后端保存仍走 `/api/option/` 的 `RootAuth()`。

校验：

- 必须是 JSON 对象。
- 模型名不能为空。
- 描述会 trim。
- 空描述会忽略，相当于恢复自动描述。

## 8. 登录与发送

已完成：

- 未登录用户可以浏览创作中心。
- 真正提交任务前会要求登录。
- 未登录点击发送会跳转到 `/sign-in`，并带上 `redirect=/creation`。
- 登录后可继续返回创作中心。

前端交互：

- Enter 发送。
- Shift + Enter 换行。
- 发送中显示 loading。
- 失败时展示上游错误信息。

## 9. 排查过的问题

### 9.1 测试渠道报 token quota 不足

错误表现：

```text
token quota is not enough
token remain quota: 1
need quota: 50
```

原因：

- 当前测试账号或渠道可用额度不足。
- 不是前端按钮问题，也不是请求格式问题。

处理方式：

- 补充额度，或切换到有余额的渠道/用户。

### 9.2 Sora 模型找不到渠道

错误表现：

```text
No available channel for model sora-2 under group default
```

原因：

- 请求模型名与后台配置模型名不一致。
- 例如前端/测试命令用 `sora-2`，而后台配置可能是 `sora2`。
- 或该模型没有配置到 `default` 分组可用渠道。

处理方式：

- 统一模型名。
- 确认渠道启用。
- 确认可用分组包含 `default`。

### 9.3 上游 auth_unavailable

错误表现：

```text
auth_unavailable: no auth available
```

原因：

- 渠道上游认证配置不可用。
- 常见是 API Key、代理认证或上游账号授权缺失。

处理方式：

- 检查渠道密钥。
- 检查上游配置。
- 检查对应渠道是否支持当前模型。

### 9.4 Zeabur 域名 404

当前表现：

```text
https://ghlink.zeabur.app/ -> HTTP/2 404
https://ghlink.zeabur.app/creation -> HTTP/2 404
```

判断：

- 代码已推送到 GitHub main。
- 旧 IP 服务仍返回 200。
- Zeabur 域名当前没有正确接入应用服务。

建议检查：

- Zeabur 服务是否运行中。
- 域名是否仍是 `PROVISIONING`。
- 绑定端口是否正确。
- 服务日志是否构建失败。
- Zeabur 项目是否拉取到最新 GitHub 提交。

## 10. 验证记录

近期已运行并通过：

```bash
/Users/mymac/.bun/bin/bunx eslint src/features/creation-center
/Users/mymac/.bun/bin/bun run typecheck
/Users/mymac/.bun/bin/bun --bun run build
git diff --check
```

i18n：

```bash
/Users/mymac/.bun/bin/bun run i18n:sync
```

结果：

- locale 文件没有 missing key。
- 新增文案已补充 `en`、`zh`、`fr`、`ja`、`ru`、`vi`。

未能本地运行：

```bash
go test ./controller -run Creation
gofmt
```

原因：

- 当前本机环境没有 `go` / `gofmt` 命令。

前端构建注意：

- 直接 `bun run build` 在 Node runtime 下遇到过 Rspack 原生绑定签名问题。
- 使用 `bun --bun run build` 可以正常构建。

## 11. 使用和维护建议

### 11.1 本地开发

推荐在 `web/default/` 下使用 Bun：

```bash
/Users/mymac/.bun/bin/bun install
/Users/mymac/.bun/bin/bun --bun run dev
/Users/mymac/.bun/bin/bun --bun run build
```

后端需要安装 Go 工具链后再运行：

```bash
go test ./...
```

### 11.2 修改后是否需要重新部署

需要区分情况：

- 只改本地文件：只影响本地，不会自动影响线上。
- 改完并提交推送到 GitHub：Zeabur 如果配置了自动部署，会自动重新部署。
- 改服务器上的运行配置：可能不需要重新构建，但通常要重启服务或刷新配置。
- 改前端页面或 Go 代码：需要重新构建和重新部署。

### 11.3 超管如何维护创作中心模型

路径：

```text
/creation
```

操作：

1. 使用超级管理员账号登录。
2. 打开创作中心。
3. 在左侧“可用模型”区域点击：
   - `管理分类`
   - `管理描述`
4. 保存后刷新模型目录，前端会重新读取 `/api/creation/models`。

### 11.4 后续建议

优先级较高：

- 修复 Zeabur 域名 404，让 `ghlink.zeabur.app` 能正常访问应用。
- 在有 Go 工具链的环境运行后端测试。
- 把创作历史从 localStorage 升级为服务端会话历史。
- 把素材库从前端临时预览升级为真实文件上传和引用。
- 为超管描述/分类管理增加独立后台菜单，避免只在创作中心侧边栏里维护。

优先级中等：

- 将 `docs/creation-center-api.md` 补充最新 `cost`、`manual_description`、分类/描述配置说明。
- 增加创作任务服务端日志页，方便追踪每次提交、任务 ID、上游响应和费用。
- 增加模型用途的批量编辑和搜索功能。

## 12. 关键文件索引

创作中心前端：

- `web/default/src/routes/creation/index.tsx`
- `web/default/src/features/creation-center/index.tsx`
- `web/default/src/features/creation-center/api.ts`
- `web/default/src/features/creation-center/session.ts`
- `web/default/src/features/creation-center/types.ts`

创作中心后端：

- `controller/creation.go`
- `controller/creation_test.go`
- `controller/option.go`
- `dto/creation.go`
- `router/api-router.go`

异步视频与 Sora：

- `controller/video_proxy.go`
- `controller/task_video.go`
- `relay/channel/task/sora/adaptor.go`
- `service/task_polling.go`

首页：

- `web/default/src/features/home/index.tsx`
- `web/default/src/features/home/components/sections/hero.tsx`
- `web/default/src/features/home/components/grainient.tsx`
- `web/default/src/features/home/components/grainient.css`

部署与说明：

- `docs/server-deploy.md`
- `docs/creation-center-api.md`
- `docker-compose.yml`
- `docker-compose.prod.yml`
- `Dockerfile`
- `Dockerfile.prod`

