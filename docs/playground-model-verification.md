# 在线模型测试记录

站点：https://ghlink.top/playground

日期：2026-10-12。验收以在线目录的 17 个模型为准；本地目录有 29 个模型，已全部返回接口参数，但参数覆盖不等于真实生成通过。

## 发布和代码检查

- 已发布最新提交：`5694689`，Zeabur 已运行该 WAN 参数修正版本。图片字段修正 `9599fb46b39a91406dffe5fc9e3598a7f947253f` 也已发布。
- 前端类型检查、相关 ESLint、生产构建通过。
- 41 项前端相关测试、11 项文档测试通过。
- 本地 Go 相关包测试通过；线上 `creation-compatibility` 的 SQLite、MySQL、PostgreSQL 三项均通过。
- 线上 Security 工作流失败，包含依赖审计与 Go 漏洞扫描报告，不能描述为全部 CI 通过。
- 第二轮 `creation-compatibility`：https://github.com/lunamoon055/new-api-N/actions/runs/38161136237 成功。Security：https://github.com/lunamoon055/new-api-N/actions/runs/38161136258 失败，尚未单独核对其详细告警。
- 文档仍然只在打开时加载，后续通过刷新按钮更新，没有定时刷新。

## 真实生成结果

| 模型 | 请求规格 | 结果 | 证据 |
| --- | --- | --- | --- |
| Grok | 6 秒、16:9、480p | 失败 | 上游返回 `no per-call price configured for resolution 480P`；网关现有固定价格为 $1.5/次，未修改定价 |
| Grok | 6 秒、16:9、720p | 通过 | `task_65gsRpKtytL4Ik7C9P1KpnkfR6Vev5jr`；视频元素实际加载为 1280×720、6.041667 秒，readyState=4，无播放错误，媒体域名 media.ghlink.top |
| gpt-image-2 | 1:1、1K、无参考图 | 生成通过，分辨率待核对 | 图片实际加载为 2048×2048，媒体域名 media.ghlink.top；不能据此声称 1K 输出规格已经验证 |
| Seedance2.0-c | 6 秒、16:9、720p、无参考素材 | 通过 | `task_omPSbiPTSFoELdLjHzFPU8e2zySuuOUQ`；视频实际 1280×720、6.08 秒、readyState=4，无播放错误 |
| gpt-image-2.5-sunburst | 1:1、1K、无参考图 | 生成通过，分辨率待核对 | 实际 2048×2048、加载完成 |
| gpt-image-2.5-flare | 1:1、1K、无参考图 | 生成通过，分辨率待核对 | 实际 2048×2048、加载完成 |
| Seedance 2.5:GZ-480P | 6 秒、16:9、480p、无参考素材 | 上游不可用 | `consumer is disabled`，上游提示每天早上 9:00 后开放；测试时间为北京时间 01:35 |
| Seedance2.0-fast-c | 6 秒、16:9、720p | 通过 | `task_F1P3j3NG6DgYbkYWn0loVJl7jK9lAx2g`；实际 1280×720、6.08 秒，无播放错误 |
| Seedance2.0-fast2-c | 6 秒、16:9、720p | 通过 | `task_ihKlmngG4L9MnHdJSAj0FyY85BCsbSvR`；实际 1280×720、6.08 秒，无播放错误 |
| nano-banana-pro | 1:1、1K、无参考图 | 通过 | 实际 1024×1024、加载完成 |
| Seedance 2.5:GZ-720P | 6 秒、16:9、720p | 上游不可用 | `consumer is disabled`，同 480P 的上游开放时间提示 |
| Seedance 2.5:GZ-1080P | 6 秒、16:9、1080p | 上游不可用 | `consumer is disabled`，同 480P 的上游开放时间提示 |
| Minimax H3:X-2k | 6 秒、16:9、2k | 通过 | `task_a5Kd2cnTU8SssRlFzjvcsqUw7tdphUSq`；实际 2560×1440、6.584 秒、readyState=4，无播放错误 |
| Seedance2.5-c | 6 秒、16:9、720p | 通过 | `task_KJ7TGKqCDuMtTAX8wMTMw48U18RNUFXb`；实际 1280×720、6.08 秒、readyState=4，无播放错误 |
| gpt-image-2 | 1:1，显式 `size=1024x1024`、`output_resolution=1K` | 通过 | 实际 1024×1024、加载完成 |
| gpt-image-2 | 1:1，显式 `output_resolution=4K`，无 size | 生成通过 | 实际 2880×2880、加载完成；上游的 4K 档不能据此宣称方图达到 4096×4096 |
| nano-banana2 | 1:1，显式 `output_resolution=4K` | 通过 | 实际 4096×4096、加载完成 |
| Seedance 2.0:X | 6 秒、16:9、720p | 通过 | `task_ZRxiJixeVvoh5u5WJeePGk4W1dPi1tw6`；实际 1280×720、6.05 秒、readyState=4，无播放错误 |
| Seedance 2.0:X1 | 6 秒、16:9、720p、海边日落 | 上游审核拒绝 | `task_aXVPQEse4WI8NTWs6xBPbHlVTJjTXM1d`：`content.policy_violation`；没有人物的自然场景仍被上游拒绝，已换纯天空描述单次复验 |
| wan3.0(图片+音频参考) | 6 秒、16:9、1 张蓝色海浪参考图，未显式设置 resolution | 通过 | `task_BrfTiAzzqeljOl7qMouhwH4UOYc0pmzz`；实际 1920×1080、6.037007 秒、readyState=4，无播放错误；参考图来自本轮生成的非人物测试插画 |
| Seedance 2.0:X1 | 6 秒、16:9、720p、纯蓝天空与白云 | 验证中 | `task_TwGJVIbjUT1GEwJ0boixDltbcOhpyhmO`，最后读取的上游状态为 running；第一次审核拒绝后仅作一次普通自然场景复验 |

测试提示词为无人物、无文字的海边日落或蓝色海浪插画。未重新生成原聊天中的其他提示词。

Grok 720p 截图：`model-integration-prototype/qa/playground-grok-720p-success.jpg`。

当前 17 个在线模型中，13 个已有实际生成输出，3 个 GZ 模型被上游暂停，X1 的第二次测试仍在处理，不能宣称全部模型通过。WAN 首次成功请求未显式设置 resolution，因此尚未证明新增清晰度选项的实际生成结果。JR 图片通道没有参数覆盖，透传请求体关闭。已按其官方文档将图片档位字段修正为 `output_resolution`，GPT Image 2.5 Flare / Sunburst 的参考图上限按当前上游目录修正为 16。

上游资料：https://linksky.top/pricing、https://my.feishu.cn/wiki/IBLkwORrriA6WSkbQjCcRcIdnfc。异步旧版文档中的 `gpt-image2` 不是当前目录的 `gpt-image-2`，未将旧模型固定 1K / 6 图规则套用到新模型。

WAN 参数按当前上游 https://suanliai.top/docs 的 `特价/wan3.0-image` 条目补齐：2–30 秒，480p/720p/1080p，10 图/0 视频/5 音频，必须发送 resolution。已发布此修正，没有调整网关或上游价格。

本地后端已更新至最新提交，用户重新登录后已确认可访问模型测试页面。未再次重启后端，避免再次使会话失效。
