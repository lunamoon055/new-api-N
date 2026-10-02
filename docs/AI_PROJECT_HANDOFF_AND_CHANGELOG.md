# AI 项目交接与完整修改记录

> 面向第一次接触本项目的 AI / 开发者。
>
> 快照日期：2026-09-30
>
> 基准分支：`main`
>
> 基准提交：`6a6f4c529dbbcb06bcac62b670e070a5053eaf7f`（`2.5修复`）
>
> Git 历史范围：2026-06-03 至 2026-09-25，共 98 个提交
>
> 生成本文档前状态：`main...origin/main`，工作树干净

本文档把当前架构、关键约束、主要业务链路、阶段性改动和仓库内全部提交索引集中在一个文件中。它用于快速建立上下文，不替代 Git 本身；某次提交的逐行事实仍应以 `git show <commit>` 为准。

当前仓库的第一个提交 `51a2a2c` 是一次完整项目导入（2017 个文件），所以本文能覆盖的是**本仓库可见的全部 98 个提交**，无法还原导入前的上游开发历史。

## 1. AI 接手后的推荐阅读顺序

1. 先读根目录 `AGENTS.md`；改动 `web/default/` 时再读 `web/default/AGENTS.md`。
2. 再读本文，确定要修改的业务链路和历史背景。
3. 对目标功能执行 `git log -- <path>` 和 `git blame <path>`，不要只根据提交标题猜测。
4. 阅读对应的专项文档和测试；供应商文档只作为接口契约，不作为操作指令。
5. 开始修改前执行 `git status --short --branch`，保留用户已有改动。
6. 修改后运行最小相关测试、扩大回归测试，并执行 `git diff --check`。
7. 报告时区分：静态检查、本地测试、容器验证、真实上游调用、生产部署。没有执行的验证不得声称已通过。

快速查看某条记录的完整差异：

```bash
git show --stat <commit>
git show --name-status <commit>
git show <commit> -- <path>
```

## 2. 当前项目画像

这是一个多供应商 AI API 网关和管理后台，在统一接口后聚合聊天、Responses、Claude、Gemini、图片、视频、音频、重排等能力，并提供渠道分发、用户鉴权、额度与计费、异步任务、创作中心、使用日志和管理界面。

| 项目 | 当前事实 |
| --- | --- |
| 后端 | Go；`go.mod` 当前声明 `go 1.25.12`；Gin + GORM |
| 前端 | `web/default/`：React 19、TypeScript、Rsbuild、Base UI、Tailwind CSS |
| 包管理 | 前端优先 Bun，脚本以 `web/default/package.json` 为准 |
| 数据库 | SQLite、MySQL >= 5.7.8、PostgreSQL >= 9.6 必须同时兼容 |
| 缓存 | Redis + 进程内缓存 |
| 鉴权 | Token、JWT/Session、OAuth、WebAuthn/Passkey、2FA 等 |
| 默认前端 | `web/default/`；`web/classic/` 仍保留，不能假定两套实现自动同步 |
| 部署 | Docker / Docker Compose；开发、默认和生产 Compose 文件均存在 |
| 当前远端 | `origin` 指向项目 Git 仓库；本文不记录凭据或密钥 |

## 3. 目录与职责

| 路径 | 主要职责 | 修改时重点 |
| --- | --- | --- |
| `router/` | HTTP 路由注册 | 认证中间件、路由顺序、兼容别名 |
| `controller/` | 请求处理与流程编排 | 参数解析、任务状态、响应结构、权限 |
| `middleware/` | 鉴权、分发、限流、协议转换 | 模型映射、渠道选择、幂等、请求体生命周期 |
| `relay/` | 统一 Relay 主流程 | 同步/异步模式、计费、请求格式转换 |
| `relay/channel/` | 各供应商适配器 | 上游 endpoint、payload、状态和错误契约 |
| `relay/channel/task/sora/` | 多种兼容型图片/视频异步任务 | `/v1/videos` 与旧 async family 必须隔离；轮询沿用提交时 endpoint |
| `relay/common/` | Relay 上下文和共享逻辑 | `RelayInfo`、渠道元数据、StreamOptions |
| `service/` | 计费、任务、错误、文件、图床等业务服务 | 事务、状态机、重试、敏感数据处理 |
| `model/` | GORM 模型和数据库访问 | 三数据库兼容、CAS/事务、迁移 |
| `setting/` | 模型、倍率、系统和性能配置 | 配置存储格式及向后兼容 |
| `pkg/billingexpr/` | 动态计费表达式引擎 | 改动前必须读 `pkg/billingexpr/expr.md` |
| `dto/` | 请求/响应 DTO | 可选标量必须用指针保留显式 `0` / `false` |
| `common/` | JSON、URL、加密、缓存等公共能力 | JSON 实际编解码统一走 `common/json.go` |
| `web/default/` | 当前 React 管理台和创作中心 | i18n、类型检查、Bun、功能模块边界 |
| `docs/` | 接口、设计、部署和集成文档 | 先判断是当前契约还是历史快照 |

整体调用关系：

```text
Router
  -> Middleware（鉴权 / 限流 / 渠道分发 / 请求转换）
  -> Controller
  -> Relay / Service
  -> Provider Adapter
  -> Model / Cache / Upstream
```

## 4. 不可破坏的项目约束

以下规则来自仓库 `AGENTS.md`，后续 AI 必须继续遵守：

1. 业务代码不得直接调用 `encoding/json` 的 Marshal/Unmarshal；使用 `common.Marshal`、`common.Unmarshal`、`common.DecodeJson` 等封装。`json.RawMessage` 等类型仍可引用。
2. 所有数据库改动同时兼容 SQLite、MySQL 和 PostgreSQL；优先使用 GORM，原生 SQL 必须处理引号、布尔值和迁移差异。
3. 前端优先使用 Bun：`bun install`、`bun run typecheck`、`bun run build`、`bun run i18n:sync`。
4. 新渠道需确认是否支持 `StreamOptions`；支持时加入 `relay/common/relay_info.go` 的 `streamSupportedChannels`。
5. 与 **new-api** 和 **QuantumNous** 有关的项目身份、品牌、署名、模块路径、许可证及元数据受保护，不得删除、替换或改名。
6. 会从客户端 JSON 解析后再转发上游的可选标量字段，必须使用带 `omitempty` 的指针类型；缺失表示省略，显式零值仍需转发。
7. 动态/阶梯计费改动前必须完整阅读 `pkg/billingexpr/expr.md`，保持表达式、预扣、结算、日志展示和版本语义一致。
8. `web/default/` 的用户可见文案需要 i18n；React 组件使用 `useTranslation()`；修改 TS/TSX 后执行类型检查。
9. 不记录或输出 API Key、Token、加密明文、签名 URL 等敏感信息；错误和日志仍需走现有脱敏逻辑。

## 5. 当前关键业务链路

### 5.1 普通同步 Relay

主要入口在 `router/relay-router.go`：

- `GET /v1/models`
- `POST /v1/chat/completions`
- `POST /v1/completions`
- `POST /v1/responses`
- `POST /v1/messages`
- `POST /v1/images/generations`
- `POST /v1/images/edits`
- 音频、Embedding、Rerank、Gemini 原生兼容等路由

请求经过 Token 鉴权、模型限流和渠道分发后进入 `controller.Relay`，再由 `relay/` 根据 RelayFormat 和渠道类型选择适配器。定位问题时要同时检查“外部公开模型名”“渠道模型映射后的上游模型名”和“实际请求 endpoint”。

### 5.2 创作中心

创作中心同时存在两类入口：

- Dashboard/Session API：`/api/creation/*`，使用用户会话鉴权。
- Token 兼容 API：`/v1/creation/*`，使用 Token 鉴权，并带创作中心幂等处理。

当前能力包括模型目录、图片生成、异步图片、异步视频、素材上传、历史/任务展示、模型分类/排序/描述、费用摘要和不同模型的输入限制。核心文件：

- `controller/creation.go`
- `controller/creation_reference_file.go`
- `middleware/creation_adapter.go`
- `middleware/creation_idempotency.go`
- `web/default/src/features/creation-center/`
- `web/default/src/features/media-generation/`

不要把 `docs/ghlink-project-summary.md` 中 2026-06-09 的早期创作中心限制当成当前实现；该文件是历史快照，之后已经加入服务端素材、更多异步模型、模型排序、幂等和大量 UI/协议修复。

### 5.3 异步图片和视频

`router/video-router.go` 当前注册的主要任务路由：

| 类型 | 提交 | 查询/内容 |
| --- | --- | --- |
| 异步图片 | `POST /v1/images/async-generations` | `GET /v1/images/async-generations/:task_id` |
| 旧视频兼容 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/:task_id` |
| 视频兼容别名 | `POST /v1/video/generations` | `GET /v1/video/generations/:task_id` |
| OpenAI 风格视频 | `POST /v1/videos` | `GET /v1/videos/:task_id` |
| 视频内容代理 | — | `GET /v1/videos/:task_id/content` |
| 视频生成别名 | `POST /v1/videos/generations` | `GET /v1/videos/generations/:task_id` |

关键不变量：

- `/v1/videos` 和 `/v1/video/async-generations` 是不同的上游协议族，不能只按“都是视频”混用。
- 提交时把实际使用的上游 endpoint 持久化为 `UpstreamEndpoint`，轮询优先复用它，避免模型名/映射变化后查询错误的协议。
- `relay/channel/task/sora/adaptor.go` 负责 endpoint 选择、请求转换、轮询、状态归一和结果转换；具体模型规则拆分在 `seedance.go`、`video2.go`、`videos.go`、`image.go`、`suanliai.go` 等文件。
- HTTP 200 或 `queued` / `in_progress` 只代表已接收或处理中，不代表生成完成。
- 临时 408/409/425/429/5xx 不应轻易当成终态失败，更不能因此重复提交一个可能已经计费的任务。

### 5.4 计费

项目同时存在多种计费模式：

- Token/倍率计费。
- 固定单次视频任务计费。
- 视频清晰度/分辨率计费（含 1K、2K 等规格）。
- `tiered_expr` 动态表达式计费。
- 订阅额度与任务结算状态。

动态表达式的主链路是：

```text
前端编辑器
  -> setting/billing_setting/tiered_billing.go 存储与校验
  -> relay/helper/price.go 预扣并冻结 BillingSnapshot
  -> service/tiered_settle.go 按实际 usage 结算
  -> service/log_info_generate.go 写入日志展示信息
```

表达式价格单位、`p/c/len/cr/cc/cc1h/img/ai/ao` 的含义、自动排除、版本和 quota 换算都以 `pkg/billingexpr/expr.md` 为准。尤其不要用预扣阶段的动态配置覆盖已冻结的结算快照。

### 5.5 图片结果、图床与视频持久化转存

同步图片结果由 `service/image_generation_result.go` 和 `service/media_storage.go` 处理，可将远程 URL 或合法 `b64_json` 结果存储并在使用日志中展示；受保护的本地结果通过 `GET /api/image-results/:filename` 获取。

视频采用更严格的持久化转存流程：

```text
上游视频任务完成
  -> 创建唯一 MediaTransferJob
  -> 加密保存上游来源和转存上下文
  -> Worker 租约领取
  -> 下载到持久化 spool
  -> 按提供商优先级上传并验证公开 URL
  -> 原子更新任务与结算状态
```

核心文件：

- `model/media_transfer_job.go`
- `service/media_transfer_worker.go`
- `service/media_transfer_crypto.go`
- `service/media_storage_video.go`
- `controller/media_transfer_admin.go`

管理员监控/重试接口：

- `GET /api/option/media_storage/transfers`
- `POST /api/option/media_storage/transfers/:id/retry`

重试只重试转存，不能重新提交生成，也不能重复计费。多实例部署需要稳定的 `CRYPTO_SECRET` 和持久化/共享的 `MEDIA_SPOOL_DIR`。

### 5.6 日志、错误和管理能力

- 视频任务错误统一经 `service.TranslateVideoTaskError` 处理，并由 `common.MaskSensitiveInfo` 等逻辑脱敏。
- 详细、可读的上游错误可以保留，但密钥、URL 凭据和敏感参数不能透传。
- 普通图片生成结果会进入绘图日志；前端任务详情同时识别视频和 `imageGenerate` 图片预览。
- 超级管理员可查看任务上传素材和完整提示词；普通用户权限必须继续隔离。
- 渠道测试实验室支持多种 endpoint，但测试 endpoint 必须与实际供应商契约一致。

## 6. 关键文档索引

| 文档 | 用途 | 时效提示 |
| --- | --- | --- |
| `AGENTS.md` | 全仓库强制规则 | 修改前必读 |
| `web/default/AGENTS.md` | 默认前端规范 | 修改默认前端时必读 |
| `pkg/billingexpr/expr.md` | 动态计费表达式完整设计 | 计费改动的权威说明 |
| `docs/creation-center-api.md` | 创作中心模型目录接口 | 结合当前路由/DTO复核 |
| `docs/async-image-model-mapping.md` | 异步图片模型映射约束 | 结合 `relay/channel/task/sora/image.go` |
| `docs/ghlink-downstream-api.md` | 下游 API 使用说明 | 发布前需与当前 Key 的 `/v1/models` 复核 |
| `docs/ghlink-downstream-codex-integration.md` | 下游任务和视频持久化说明 | 结合当前部署配置复核 |
| `docs/ghlink-latest-api-integration.md` | 模型/图床集成记录 | 属于专项历史文档 |
| `docs/ghlink-seedance2-video-api.md` | Seedance 视频说明 | 供应商契约可能变化，接入前重验 |
| `docs/ghlink-project-summary.md` | 2026-06-09 的早期项目总结 | 明确是旧快照，不代表当前全貌 |
| `docs/server-deploy.md` | 服务端部署 | 使用前核对当前 Compose/环境变量 |

## 7. 阶段性改动摘要

### 阶段 A：完整项目导入与创作中心起步（2026-06-03 ～ 2026-06-09）

- `51a2a2c` 导入完整网关、管理后台、适配器、计费、认证、Docker、文档和测试基础。
- `470fd2a`、`74c0d5d` 建立创作中心异步媒体/视频提交、轮询和前端会话链路。
- `3a3764f` 让默认前端正确服务 `/creation`。
- `44cb7b0`、`fa2f409` 调整首页并接入 Grainient 视觉。
- `e5061b7`、`5ead4c9`、`7b74093`、`85e8ef1` 完善模型分类、素材发送、零消耗展示和模型描述管理。

### 阶段 B：媒体路由、视频预览与固定计费（2026-06-17 ～ 2026-06-21）

- 修复 Playground 媒体路由，加入异步视频预览和 LinkSky 模型支持。
- 修正 Sora2 参数；加入视频固定费用设计和实现。
- 为 Video2 增加请求校验、创作能力和参考素材。
- 重构创作中心 UI、修复参考图 URL、DT 回传和 API 信息泄漏问题。
- `24175f5` 的渠道分组/模型匹配规范化随后由 `0ff0697` 回退；判断当前行为必须看回退后的代码，而不是只看前一个提交。
- `717b8af` 将 classic 的相关能力同步到 default 前端。

### 阶段 C：创作中心素材、价格与渠道测试（2026-06-25 ～ 2026-07-09）

- 加入服务端创作素材文件处理、更多 Video2/Claude 流式适配和前端参考素材能力。
- 修复 DT 视频回传、Image2 参考图/上传/下载问题，并精简部分语言资源。
- 加入视频预览测速、任务提示词展示、分组定价修复和任务分组计费修复。
- 建立渠道 API 测试实验室与前端 endpoint 选择。
- 加入视频分辨率价格、动态阶梯价格展示和结算逻辑。
- 新增“三宝”任务渠道适配器、渠道测试、创作中心输入限制和结果解析。

### 阶段 D：管理台、统计、供应商扩展与安全加固（2026-07-20 ～ 2026-08-13）

- 多次优化创作中心 UI、成功/失败计数和 Dashboard 展示。
- 接入 Mega 视频渠道并修复其测试 endpoint。
- `073db40` 大范围强化配额预留、任务计费事件、订阅结算、会话/2FA、可信代理和前端体验。
- `307ce31` 增强多参考图视频、SSRF 防护、安全下载和支付 webhook 安全。
- `426a09f` 进行 CI、依赖、OAuth、支付、日志和多渠道适配器的大范围修复。
- 加入 SD2/933、Seedance 2.5、Video2.5 等模型的前后端能力与素材限制。
- 超级管理员可查看上传素材；任务中心增加筛选；修复成功条数算法。
- 建立统一任务错误翻译/恢复逻辑，之后调整为保留可读上游错误并继续脱敏。

### 阶段 E：幂等、轮询可靠性和错误透传（2026-08-25 ～ 2026-09-01）

- `9eb1670` 为创作中心 Token 兼容 API 增加幂等记录，并修复映射后的 Seedance `/v1/videos` 嵌套请求。
- `189cf4f` 补齐 MiniMax H3/Wan 异步识别；临时轮询错误不再轻易变成终态失败和退款；完善下载兜底。
- `c99dc87` 让详细、可读的上游错误保留原意，同时继续执行敏感信息遮蔽。

### 阶段 F：模型元数据、分辨率计费、绘图日志和媒体持久化（2026-09-09 ～ 2026-09-20）

- 模型广场加入可管理的模型备注/描述。
- 完善 1K/2K 分辨率价格、前端配置和实际计费链路。
- 修正图片 endpoint 类型识别、请求校验和错误处理。
- 普通图片生成结果进入绘图日志，支持 URL 和受控 `b64_json` 文件。
- 接入算力类视频 API，新增图床配置、图片结果重写和视频代理支持。
- 建立持久化视频转存任务、加密上下文、worker、租约、CAS 状态更新、管理员监控与手动重试。
- 转存失败不重新生成、不重复扣费；生产环境需要稳定密钥和持久 spool。

### 阶段 G：下游文档、异步图片与精确模型映射（2026-09-21 ～ 2026-09-25）

- 定价页增加模型 endpoint 名称/预设展示和下游使用说明。
- `aa4642a` 建立异步图片模型链路、渠道测试、创作中心/Playground 路由和下游 API 文档。
- `3d37469` 修正图片渠道选择、LinkSky 判断、URL 校验和渠道能力过滤。
- `2d33389` 将异步图片协议选择绑定到稳定的公开/映射模型关系，并补充轮询测试。
- `0a565d9` 修复任务日志未识别 `imageGenerate` 图片预览。
- `6a6f4c5` 对 meaicc Seedance 映射采用“精确 Base URL 主机 + 映射后上游模型”判断，修复 Seedance2.5-c 嵌套 `/v1/videos` payload，并保持其他渠道隔离。

## 8. 完整 Git 提交索引（98 条）

说明：表格保留原始提交标题，文件数是该提交直接变更的文件数量。合并提交 `f769880` 本身没有额外直接 diff，其内容来自被合并提交。标题较短或含义不完整时，应使用 `git show` 阅读实际差异。

### 2026-06（40 条）

| 日期 | 提交 | 原始说明 | 文件数 |
| --- | --- | --- | ---: |
| 06-03 | `51a2a2c` | first commit | 2017 |
| 06-03 | `470fd2a` | 异步 | 24 |
| 06-04 | `74c0d5d` | Restore creation center async video fixes | 16 |
| 06-04 | `3a3764f` | Serve creation center from default frontend | 6 |
| 06-05 | `44cb7b0` | 主页 | 2 |
| 06-05 | `fa2f409` | Integrate Grainient hero and simplify home page | 7 |
| 06-05 | `e5061b7` | 创作中心筛选 | 12 |
| 06-05 | `5ead4c9` | 完善创作中心模型分类和素材发送 | 13 |
| 06-05 | `7b74093` | 修正创作中心零消耗展示 | 3 |
| 06-06 | `85e8ef1` | 添加创作中心模型描述管理 | 13 |
| 06-09 | `10efd91` | Refine commit message generation prompt | 1 |
| 06-17 | `31903a8` | 修复游乐场媒体模型接口路由 | 6 |
| 06-17 | `1269c20` | 支持异步视频预览 | 17 |
| 06-17 | `2aa3901` | 修复异步视频预览链路 | 9 |
| 06-17 | `a7450c5` | 补齐 linksky 异步视频模型支持 | 6 |
| 06-19 | `07f8ff0` | 将游乐场文案改为模型测试 | 2 |
| 06-19 | `34e7edd` | 修复 Sora2 视频模型参数 | 5 |
| 06-19 | `bdf25d6` | docs: 设计视频固定费用计费模式 | 1 |
| 06-19 | `016a361` | feat: support fixed video task billing | 25 |
| 06-19 | `f769880` | Merge pull request #1 from lunamoon055/codex-video-fixed-fee-billing | 0 |
| 06-19 | `95faaee` | docs: design Video2 async creation options | 1 |
| 06-19 | `eafcdda` | chore: ignore local worktrees | 1 |
| 06-19 | `50a212d` | feat: validate Video2 async requests | 3 |
| 06-19 | `8212b4d` | feat: add Video2 creation capabilities | 6 |
| 06-20 | `158e93e` | feat: complete Video2 creation references | 4 |
| 06-20 | `0ccc27f` | 创作中心ui重构 | 13 |
| 06-20 | `b38d82b` | 参考图片修正 | 13 |
| 06-20 | `273745d` | 参考图片url | 17 |
| 06-20 | `d48d895` | DT Url修复,创作中心显示修复 | 5 |
| 06-20 | `35c8dfe` | api 泄漏 | 1 |
| 06-21 | `24175f5` | Normalize channel groups and model matching | 5 |
| 06-21 | `0ff0697` | 返回 | 5 |
| 06-21 | `717b8af` | Sync default frontend with classic changes | 22 |
| 06-25 | `baf8244` | 创作中心修改 | 23 |
| 06-26 | `f925105` | DT视频回传BUG修复 | 14 |
| 06-26 | `faedf42` | image2增加参考图入口 | 14 |
| 06-26 | `34a5713` | image2上传问题修复 | 15 |
| 06-26 | `a970362` | 下载错误及删除中英文以外的语言 | 4 |
| 06-29 | `4f4ad21` | 视频预览加测速修复 | 19 |
| 06-29 | `44914af` | 提示词队列，及分组定价bug修复 | 12 |

### 2026-07（21 条）

| 日期 | 提交 | 原始说明 | 文件数 |
| --- | --- | --- | ---: |
| 07-03 | `cc88793` | fix creation task group pricing | 9 |
| 07-04 | `6b4dbf6` | image2修正 | 6 |
| 07-06 | `2aa317d` | api测试渠道 | 11 |
| 07-06 | `40d29e4` | @参考构建 | 6 |
| 07-08 | `42eddf5` | 增加前端模型接口 | 15 |
| 07-09 | `7bddad9` | 增加阶梯价格 | 46 |
| 07-09 | `3312040` | 三宝适配 | 18 |
| 07-09 | `61a17d5` | 三宝前端接口增加及限制 | 19 |
| 07-09 | `3391d3a` | 修复创作中心没显示对应选项 | 6 |
| 07-09 | `511698b` | 前端显示修复3 | 7 |
| 07-09 | `4d95fb1` | 生成通道修复 | 7 |
| 07-20 | `442544e` | 创作中心ui修改 | 9 |
| 07-24 | `8fc842f` | 成功条数跟踪 | 11 |
| 07-24 | `9b28390` | 失败条数修正 | 9 |
| 07-28 | `0c59d3d` | 接入mega | 23 |
| 07-28 | `b05e07f` | 修复 Mega 视频模型测试端点 | 7 |
| 07-29 | `073db40` | Improve dashboard functionality and frontend user experience | 75 |
| 07-29 | `307ce31` | 4图模型配对 | 32 |
| 07-30 | `426a09f` | bug修复 | 89 |
| 07-30 | `c0e0a19` | 生成界面优化 | 12 |
| 07-30 | `b838b13` | fix(ci): prepare embedded frontend assets for Go checks | 1 |

### 2026-08（16 条）

| 日期 | 提交 | 原始说明 | 文件数 |
| --- | --- | --- | ---: |
| 08-04 | `2f2e6d9` | sd2,933模型前端配对 | 3 |
| 08-04 | `79ce393` | 超级管理员查看上传素材 | 18 |
| 08-05 | `ba9e8c4` | h3+500错误修复 | 26 |
| 08-08 | `4f9a125` | (线路3)sd-2.0-933前端接口调试 | 24 |
| 08-08 | `047d914` | 2.5模型对接 | 19 |
| 08-09 | `832b867` | 登录名称限制，及任务中心筛选功能 | 21 |
| 08-09 | `cc08902` | 成功条数 = max(0, 总调用数 − 失败条数) | 9 |
| 08-09 | `f506cd9` | 创作中心ui优化 | 6 |
| 08-09 | `f9abe34` | 2.5素材限制 | 10 |
| 08-10 | `95b2b56` | video2.5前端端口设置 | 17 |
| 08-10 | `24b04fa` | 模型排序 | 12 |
| 08-11 | `acfdea5` | 报误翻译 | 22 |
| 08-12 | `ad6e16f` | 报错翻译2 | 3 |
| 08-13 | `f512625` | Fix Seedance 2.5 references and recover historical task errors | 11 |
| 08-25 | `9eb1670` | (线路3)sd-2.0-fast创作中心添加接口 | 18 |
| 08-31 | `189cf4f` | MiniMax H3/Wan 异步轮询、临时错误与内容下载兜底修复 | 13 |

> `189cf4f` 的原始提交标题是一段较长的问题描述；这里为了表格可读性做了等义压缩。精确原文请执行 `git show -s --format=%s 189cf4f`。

### 2026-09（21 条）

| 日期 | 提交 | 原始说明 | 文件数 |
| --- | --- | --- | ---: |
| 09-01 | `c99dc87` | 详细、可读的上游错误会原文透传，不再翻译成通用中文。 | 6 |
| 09-09 | `9e3f55a` | 添加模型备注 | 11 |
| 09-14 | `0ca6849` | 1k,2k | 14 |
| 09-14 | `0d1a2e6` | 定价 | 5 |
| 09-15 | `8813124` | 定价链路修复 | 6 |
| 09-16 | `09106d9` | 分辨率修复 | 3 |
| 09-18 | `0bdd870` | 接口修复 | 8 |
| 09-18 | `318db4a` | 绘图日志 | 22 |
| 09-19 | `0bd67c5` | 对接算力ai | 9 |
| 09-19 | `f72828d` | 图床及对接模型 | 35 |
| 09-20 | `d59f9c4` | 测试url | 12 |
| 09-20 | `4db7651` | 转存 | 8 |
| 09-20 | `bf2d301` | 算力转存修复 | 13 |
| 09-20 | `61ba9d9` | 修复视频图床链接 | 3 |
| 09-20 | `d5b8a07` | 手动退款 | 24 |
| 09-21 | `c5e61da` | 模型名称 | 10 |
| 09-21 | `aa4642a` | 图片模型对接 | 50 |
| 09-22 | `3d37469` | 图片接口修正 | 21 |
| 09-23 | `2d33389` | 模型修复 | 11 |
| 09-24 | `0a565d9` | 已修复。根因是任务日志详情列只处理视频预览，没有识别 imageGenerate 图片任务。 | 4 |
| 09-25 | `6a6f4c5` | 2.5修复 | 3 |

## 9. 当前已知风险与验证边界

1. **供应商契约易漂移。** 接入或修复前必须重新核对真实 Base URL、公开模型名、映射后上游模型名、payload 和轮询 endpoint。
2. **不要根据模型名猜 endpoint。** 任务提交后应持久化 `UpstreamEndpoint` 并在轮询时复用。
3. **异步受理不等于完成。** `queued`、`in_progress` 和 HTTP 200 不能作为最终结果证据。
4. **不能重复产生付费任务。** 提交超时或临时轮询错误时，先查询原任务；仅在明确失败且契约允许时重试提交。
5. **图片与视频的存储保证不同。** 视频有持久化 transfer job；图片上传失败时是否允许回退原 URL，应按当前安全要求重新审计 `service/media_storage.go` 的每条分支。
6. **部署配置是视频转存正确性的一部分。** 多实例/重启场景必须验证稳定 `CRYPTO_SECRET`、共享 `MEDIA_SPOOL_DIR`、数据库迁移和 worker 生命周期。
7. **历史测试结果不是当前生产证明。** 最近的 Seedance 修复有聚焦 Go 测试记录，但没有付费上游生成或生产部署；持久化转存有本地 Go/前端验证记录，但没有真实生产大文件转存。
8. **旧文档可能滞后。** 特别是 `docs/ghlink-project-summary.md`；若文档与路由、DTO、测试冲突，以当前代码和当前上游契约为准，并同步修正文档。
9. **提交历史包含大范围混合提交。** `073db40`、`426a09f`、`aa4642a` 等不能靠标题判断影响范围，修改相关模块前应逐文件查看。
10. **仓库曾提交临时/二进制资料。** 后续不要继续提交 `.swp`、网络抓包、临时输出、私密接口文档或包含凭据的文件。

## 10. 后续 AI 修改清单

### 开始前

```bash
git status --short --branch
git log -10 --oneline
rg --files -g 'AGENTS.md'
git log --oneline -- <目标路径>
```

- 确定改动属于同步 Relay、异步任务、创作中心、计费、媒体存储还是前端展示。
- 找到现有测试和相邻供应商适配器，但不要复制错误的 endpoint 族。
- 如果涉及模型映射，同时记录公开模型 ID、上游模型 ID、渠道类型和 Base URL。
- 如果涉及数据库，设计 SQLite/MySQL/PostgreSQL 三套兼容行为。

### 修改中

- 仅改任务范围内文件，保留工作树中的其他改动。
- DTO 可选数值/布尔值使用指针，测试“缺失”和“显式 0/false”两个场景。
- endpoint、请求体、轮询和结果解析放在正确适配器内，并为其他渠道增加隔离测试。
- 生命周期和计费更新使用已有事务/CAS 模式，避免重复扣费、重复退款或旧轮询覆盖新状态。
- 前端新增文案同步 i18n；新增模型能力同时核对创作中心、Playground、渠道测试和定价详情。

### 验证

按改动范围选择，不能把未执行命令写成通过：

```bash
# Go 聚焦测试示例
go test ./relay/channel/task/sora ./relay/common ./relay -count=1

# 全量后端（环境允许时）
go test ./...

# 默认前端
cd web/default
bun run typecheck
bun run lint
bun run build

# 回到仓库根目录
git diff --check
git status --short
```

真实供应商调用、生产部署、数据库迁移、对象存储上传和大文件转存需要单独授权与真实环境；本地单元测试不能替代这些验证。

## 11. 以后如何维护本文档

每次完成较大改动后，在“阶段性改动摘要”追加一条，并在下面模板中记录事实：

```markdown
### YYYY-MM-DD：改动名称

- 提交：`<hash>`（如尚未提交，写“未提交”）
- 原因：触发问题或需求
- 影响范围：路由 / Controller / Service / Model / Adapter / Frontend
- 行为变化：外部可观察到的变化
- 兼容性：其他渠道、数据库、旧任务、旧配置是否受影响
- 验证：实际执行的命令与结果
- 未验证：生产部署、真实上游、付费任务、对象存储等
- 回滚关注点：数据迁移、配置、任务状态或缓存
```

若需要重新生成完整提交索引，可使用：

```bash
git rev-list --count HEAD
git log --reverse --date=short --format='%ad|%h|%s'
```
