# XJZ 视频模型增补接入（2026-10-14）

对照[供应商文档](https://api.xjzapi.cc/docs?model=grok-1.5-700)及[公开模型目录](https://api.xjzapi.cc/api/pricing)，新增 25 个精确视频 ID。当前专门适配的总数为 38，原有 13 个 ID 及其约束保留。这里只包含视频模型。

当前公开目录中的 19 个视频 ID 均已覆盖。本次增补的其中 16 个来自该目录；其余 9 个来自文档附录。附录 ID 是否对当前 API Key 开放，仍以供应商当前线路和本地渠道配置为准。

## 新增模型

| 精确模型 ID | 时长（秒） | 分辨率 | 计费单位 | 图片 / 视频 / 音频上限 | 图片字段 | 视频字段 | 音频字段 |
|---|---|---|---|---|---|---|---|
| `minimax-h3-2K-933` | 4–15 | 2k | 按次 | 9 / 3 / 3 | `reference_images` | `reference_videos` | `reference_audios` |
| `sd2.5-全参过真人-30-10-10` | 5–30 | 480p / 720p / 1080p | 按秒 | 30 / 10 / 10 | `reference_images` | `reference_videos` | `reference_audios` |
| `sd2.0-全参过真人-720p` | 5–15 | 720p | 按秒 | 9 / 3 / 3 | `reference_images` | `reference_videos` | `reference_audios` |
| `grok-1.5-700` | 3–15 | 720p | 按次 | 7 / 0 / 0 | `reference_images` | 不支持 | 不支持 |
| `dola-sd-2.5` | 5–30 | 480p / 720p / 1080p | 按次 | 30 / 10 / 10 | `reference_images` | `reference_videos` | `reference_audios` |
| `dola-sd-2.0-900` | 4–15 | 720p | 按次 | 9 / 0 / 0 | `reference_images` | 不支持 | 不支持 |
| `dola-sd-2.0` | 4–15 | 720p | 按次 | 9 / 0 / 3 | `reference_images` | 不支持 | `reference_audios` |
| `sd-2.0-933-720-fast-原生真人` | 4–15 | 720p | 按次 | 9 / 3 / 3 | `reference_images` | `reference_videos` | `reference_audios` |
| `gz-sd2.5-1080p` | 4–30 | 1080P | 按秒 | 30 / 10 / 10 | `images` | `reference_videos` | `reference_audios` |
| `gz-sd2.5-720p` | 4–30 | 720P | 按秒 | 30 / 10 / 10 | `images` | `reference_videos` | `reference_audios` |
| `gz-sd2.5-480p` | 4–30 | 480P | 按秒 | 30 / 10 / 0 | `images` | `reference_videos` | 不支持 |
| `gz-sd4k` | 4–15 | 4K | 按秒 | 9 / 3 / 3 | `images` | `reference_videos` | `reference_audios` |
| `gz-sd1080p` | 4–15 | 1080P | 按秒 | 9 / 3 / 3 | `images` | `reference_videos` | `reference_audios` |
| `gz-sd720p` | 4–15 | 720p | 按秒 | 9 / 3 / 3 | `images` | `reference_videos` | `reference_audios` |
| `gz-sd480p` | 4–15 | 480p | 按秒 | 9 / 3 / 3 | `images` | `reference_videos` | `reference_audios` |
| `sd2.5-720p-ch1` | 30 | 720p | 按次 | 9 / 0 / 0 | `images` | 不支持 | 不支持 |
| `viraldance900` | 4–15 | 720p | 按次 | 9 / 0 / 0 | `image_urls` | 不支持 | 不支持 |
| `viraldance933` | 4–15 | 720p | 按次 | 9 / 3 / 3 | `image_urls` | `video_urls` | `audio_urls` |
| `viraldance933-fast` | 4–15 | 720p | 按次 | 9 / 3 / 3 | `image_urls` | `video_urls` | `audio_urls` |
| `viraldance933x-fast` | 4–14 | 720p | 按秒 | 1 / 1 / 1 | `image_url` | `video_url` | `audio_url` |
| `viraldance2.5-15-stable` | 4–15 | 720p | 按次 | 30 / 0 / 3 | `image_urls` | 不支持 | `audio_urls` |
| `viraldance2.5-480p-15-stable` | 4–15 | 480p | 按次 | 30 / 0 / 3 | `image_urls` | 不支持 | `audio_urls` |
| `viraldance2.5x-30` | 4–30 | 720p | 按秒 | 30 / 0 / 3 | `image_urls` | 不支持 | `audio_urls` |
| `viraldance2.5x-480p-30` | 4–30 | 480p | 按秒 | 30 / 0 / 3 | `image_urls` | 不支持 | `audio_urls` |
| `quanneng2.0` | 15 | 720P | 按次 | 9 / 3 / 3 | `images` | `reference_videos` | `reference_audios` |

## 请求字段

- 全部模型使用本网关的 `POST /v1/videos` 创建任务，使用 `GET /v1/videos/{task_id}` 轮询；完成后可使用 `GET /v1/videos/{task_id}/content` 下载。请求头使用本网关签发的 `Authorization: Bearer <API_KEY>`。
- `viraldance900` 使用 `aspect_ratio` 与 `size`。
- `viraldance933x-fast` 使用 `aspect_ratio` 与 `resolution`，三个参考素材字段均为单个 URL 字符串。
- 其余新增模型使用 `ratio` 与 `resolution`；素材字段按上表传 URL 数组。
- `viraldance2.5-15-stable` 和 `viraldance2.5-480p-15-stable` 的时长省略时默认为 15 秒；`sd2.5-720p-ch1` 固定为 30 秒，`quanneng2.0` 固定为 15 秒。
- `grok-1.5-700` 为 3–15 秒；旧的 `TTP-grok` 保持 6–15 秒，两个 ID 不互用。
- `sd2.5-全参过真人-30-10-10` 与 `dola-sd-2.5` 支持 480p / 720p / 1080p，5–30 秒；与旧固定 30 秒 Dola 模型分开处理。
- `gz-sd480p`、`gz-sd720p`、`gz-sd1080p`、`gz-sd4k` 使用音频时必须有图片，只有视频不能替代图片。含参考视频的请求按整单乘 2，传多条视频不叠加倍率。
- `viraldance933` 和 `viraldance933-fast` 的音频需要图片或视频搭配。
- 图片、视频和音频 URL 必须可公开下载，单个文件最多 10 MB。`generate_audio: false` 会原样保留。

当前目录的 `gz-sd2.5-480p` 描述提到音频，但结构化 `input_fields` 与 `reference_limits` 没有音频能力；本次遵循结构化配置，仅启用图片和视频。后续供应商补齐明确参数后再开放音频。

## 最小示例

```json
{
  "model": "grok-1.5-700",
  "prompt": "一只橘猫在窗边抬头，镜头轻微推进",
  "duration": 3,
  "ratio": "16:9",
  "resolution": "720p"
}
```

```json
{
  "model": "viraldance933x-fast",
  "prompt": "保持参考人物一致，按音频节奏完成动作",
  "duration": 8,
  "aspect_ratio": "9:16",
  "resolution": "720p",
  "image_url": "https://example.com/reference.jpg",
  "audio_url": "https://example.com/reference.mp3",
  "generate_audio": false
}
```

## 配置与验证边界

管理员需在渠道中启用精确模型 ID，并配置本地售价；代码不会复制供应商售价或改动线上渠道。按次模型使用 `fixed` 或 `tiered_request`，按秒模型使用按秒定价（多分辨率可用 `tiered_seconds`）。未配置价格的 XJZ 模型会在预扣前拒绝；按次模型不能使用按秒计费模式。

提交成功只代表任务受理。`unknown` / `queued` / `processing` / `in_progress` 应继续等待，明确失败再处理退款。回归测试使用本地构造的请求和响应，不会发起供应商付费生成，不能替代真实渠道的端到端验收。
