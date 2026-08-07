# GHLINK 全模型对接文档增补稿

> 原文档：[异步媒体 API 下游调用文档示例](https://docs.qq.com/aio/DTkNKaWZ6ZVZVc1FJ)
>
> 使用方式：原文档第 1～9 章保持不变，将本文件第 10 章起的内容直接追加到原文档末尾。本文不替换、不删改原文档已有的接口、参数、响应、轮询和错误码说明，只补充模型广场当前新增的 11 个模型。
>
> 模型核对时间：2026-08-05。模型名称必须按本文原样传递，包括大小写、连字符、数字和中文括号。

## 全部模型范围

合并后文档共覆盖 21 个模型。

### 原文档已有模型（10 个，内容不变）

| 类型 | 模型 |
| --- | --- |
| 视频 | `sora2` |
| 视频 | `veo31` |
| 视频 | `veo31-fast` |
| 视频 | `veo31-ref` |
| 视频 | `kling-v3` |
| 视频 | `grok-imagine-video` |
| 图片 | `nano-banana` |
| 图片 | `nano-banana2` |
| 图片 | `nano-banana-pro` |
| 图片 | `gpt-image2` |

### 本次新增模型（11 个）

| 模型 | 类型 | 提交接口 | 查询接口 |
| --- | --- | --- | --- |
| `minimax-h3` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `sd2-1080P(933按秒)` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `sd2-4k(933按秒)` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `sd2-720P(933)` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0-480p` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0-fast` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0-fast-480p` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0-mini` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `video-2.0-mini-480p` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |
| `videos-4（4图3视频1音频）` | 视频 | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` |

---

## 10. 新增模型通用接入说明

Base URL、认证方式和异步调用流程继续使用原文档第 1 章的内容：

```text
Base URL: https://ghlink.top
Authorization: Bearer sk-你的令牌
Content-Type: application/json
```

统一流程不变：

1. 调用 `POST /v1/video/async-generations` 提交任务。
2. 从响应中保存 `task_id`。
3. 每 3～5 秒调用一次 `GET /v1/video/async-generations/{task_id}`。
4. `status` 为 `completed` 后读取结果 URL。

原文档第 5～9 章中的提交响应、查询响应、JS 轮询、接入建议和通用错误码同样适用于本次新增模型。

## 11. MiniMax H3

### 11.1 `minimax-h3`

文生视频：

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "minimax-h3",
    "prompt": "一位宇航员走在红色沙漠上，远处是巨大的蓝色行星，电影级光影，镜头缓慢推进",
    "duration": 8,
    "aspect_ratio": "16:9"
  }'
```

单图图生视频：

```json
{
  "model": "minimax-h3",
  "prompt": "让画面中的人物自然转身并看向镜头，保持人物身份和场景不变",
  "duration": 8,
  "aspect_ratio": "16:9",
  "image_url": "https://example.com/character.png"
}
```

多图参考：

```json
{
  "model": "minimax-h3",
  "prompt": "使用图一的人物和图二的场景，生成连续自然的电影镜头",
  "duration": 10,
  "aspect_ratio": "16:9",
  "image_urls": [
    "https://example.com/character.png",
    "https://example.com/scene.png"
  ]
}
```

首尾帧视频：

```json
{
  "model": "minimax-h3",
  "prompt": "让人物从第一帧的站姿自然过渡到最后一帧的奔跑姿态",
  "duration": 8,
  "aspect_ratio": "16:9",
  "start_image_url": "https://example.com/first-frame.png",
  "end_image_url": "https://example.com/last-frame.png"
}
```

图片加音频参考：

```json
{
  "model": "minimax-h3",
  "prompt": "让图中角色跟随音乐自然舞动，动作与节奏同步",
  "duration": 8,
  "aspect_ratio": "4:3",
  "image_urls": [
    "https://example.com/character.png",
    "https://example.com/scene.png"
  ],
  "audio_url": "https://example.com/music.ogg"
}
```

说明：

- `prompt` 最长 2000 个字符。
- `duration` 支持 5～15 秒。
- `aspect_ratio` 支持 `16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9`。
- 输出分辨率固定为 2K，请勿额外传 `resolution`。
- 最多 5 张参考图片，不支持参考视频。
- 最多 1 个参考音频；音频时长建议 2～15 秒。
- 参考音频支持 MP3、WAV、M4A、AAC、OGG、WebM。
- 使用音频时必须同时提供至少一张参考图片。
- 首尾帧模式必须同时传 `start_image_url` 和 `end_image_url`。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `minimax-h3` |
| `prompt` | string | 是 | 视频生成提示词，最长 2000 个字符 |
| `duration` | number | 否 | 5～15 秒 |
| `aspect_ratio` | string | 否 | `16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9` |
| `image_url` | string | 否 | 单张参考图 URL |
| `image_urls` | array | 否 | 多张参考图 URL，最多 5 张 |
| `start_image_url` | string | 否 | 首帧图片 URL |
| `end_image_url` | string | 否 | 尾帧图片 URL |
| `audio_url` | string | 否 | 参考音频 URL，必须与参考图片一起使用 |

## 12. SD2 系列

### 12.1 模型规格

| 模型 | 时长 | 比例 | 分辨率传参 |
| --- | --- | --- | --- |
| `sd2-1080P(933按秒)` | 4～15 秒 | `16:9`、`9:16`、`1:1` | 固定 1080P，请勿传 `resolution` |
| `sd2-4k(933按秒)` | 4～15 秒 | `16:9`、`9:16`、`1:1` | 固定 4K，请勿传 `resolution` |
| `sd2-720P(933)` | 4～15 秒 | `16:9`、`9:16`、`1:1` | 传 `720p` 或 `480p` |

### 12.2 `sd2-1080P(933按秒)`

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2-1080P(933按秒)",
    "prompt": "清晨海边公路，跑车沿海岸线行驶，电影级航拍，光影自然",
    "duration": 8,
    "ratio": "16:9"
  }'
```

### 12.3 `sd2-4k(933按秒)`

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2-4k(933按秒)",
    "prompt": "雪山脚下的湖面倒映晚霞，镜头从湖面缓慢升高，电影级自然风光",
    "duration": 10,
    "ratio": "16:9"
  }'
```

### 12.4 `sd2-720P(933)`

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2-720P(933)",
    "prompt": "一位滑板少年穿过霓虹街区，低机位跟拍，动作流畅",
    "duration": 6,
    "ratio": "9:16",
    "resolution": "720p"
  }'
```

### 12.5 SD2 多媒体参考

三个 SD2 模型使用同一组参考素材字段。固定 1080P/4K 模型仍然不要传 `resolution`。

```json
{
  "model": "sd2-720P(933)",
  "prompt": "参考图片中的人物、参考视频的运镜和参考音频的节奏，生成一段连续广告视频",
  "duration": 8,
  "ratio": "16:9",
  "resolution": "720p",
  "referenceImages": [
    "https://example.com/person.png",
    "https://example.com/product.png"
  ],
  "referenceVideos": [
    "https://example.com/camera-motion.mp4"
  ],
  "referenceAudios": [
    "https://example.com/music.mp3"
  ]
}
```

说明：

- `prompt` 不能为空，最长 5000 个字符。
- `duration` 支持 4～15 秒。
- `ratio` 只支持 `16:9`、`9:16`、`1:1`。
- 最多 9 张参考图片、3 个参考视频、3 个参考音频，全部参考素材合计最多 15 个。
- 参考素材必须使用公网可访问的 HTTP/HTTPS URL。
- 不支持 `first_image` 和 `last_image`；首尾帧图片应放入 `referenceImages` 并在 `prompt` 中说明顺序。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 三个 SD2 模型名之一，必须原样传递 |
| `prompt` | string | 是 | 视频提示词，最长 5000 个字符 |
| `duration` | number | 否 | 4～15 秒 |
| `ratio` | string | 否 | `16:9`、`9:16`、`1:1` |
| `resolution` | string | 否 | 仅 `sd2-720P(933)` 使用，传 `720p` 或 `480p` |
| `referenceImages` | array | 否 | 参考图片 URL，最多 9 张 |
| `referenceVideos` | array | 否 | 参考视频 URL，最多 3 个 |
| `referenceAudios` | array | 否 | 参考音频 URL，最多 3 个 |

## 13. Video 2.0 系列

### 13.1 模型规格

| 模型 | 固定分辨率 | 可用尺寸 |
| --- | --- | --- |
| `video-2.0` | `720p` | `720x1280`、`1280x720`、`960x960` |
| `video-2.0-fast` | `720p` | `720x1280`、`1280x720`、`960x960` |
| `video-2.0-mini` | `720p` | `720x1280`、`1280x720`、`960x960` |
| `video-2.0-480p` | `480p` | `496x864`、`864x496`、`640x640` |
| `video-2.0-fast-480p` | `480p` | `496x864`、`864x496`、`640x640` |
| `video-2.0-mini-480p` | `480p` | `496x864`、`864x496`、`640x640` |

六个模型的请求格式一致，只需要替换 `model`。720p 模型必须传 `resolution: "720p"`；名称以 `-480p` 结尾的模型必须传 `resolution: "480p"`。

### 13.2 文生视频

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "video-2.0-fast",
    "prompt": "一辆红色跑车在雨夜城市中穿行，霓虹反射，电影级追拍镜头",
    "duration": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p",
    "async": true
  }'
```

480p 模型：

```json
{
  "model": "video-2.0-fast-480p",
  "prompt": "一只橘猫在阳光下的窗台伸懒腰，镜头缓慢靠近",
  "duration": 6,
  "aspect_ratio": "9:16",
  "resolution": "480p",
  "async": true
}
```

### 13.3 图片参考

```json
{
  "model": "video-2.0",
  "prompt": "保持参考图主体和构图，让人物自然抬头并看向远方",
  "duration": 8,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "async": true,
  "image_urls": [
    "https://example.com/person.png",
    "https://example.com/scene.png"
  ]
}
```

### 13.4 首尾帧、视频和音频参考

```json
{
  "model": "video-2.0-fast",
  "prompt": "让角色从首帧姿态自然运动到尾帧姿态，参考视频的镜头语言并匹配音频节奏",
  "duration": 10,
  "aspect_ratio": "16:9",
  "resolution": "720p",
  "async": true,
  "start_image_url": "https://example.com/first-frame.png",
  "end_image_url": "https://example.com/last-frame.png",
  "video_reference": [
    { "url": "https://example.com/camera-1.mp4" },
    { "url": "https://example.com/camera-2.mp4" }
  ],
  "audio_url": "https://example.com/music.mp3"
}
```

说明：

- `prompt` 不能为空，最长 5000 个字符。
- `duration` 支持 4～15 秒。
- `aspect_ratio` 支持 `9:16`、`16:9`、`1:1`。
- `size` 与 `aspect_ratio + resolution` 是两种输出规格表达方式；建议只使用后一种。
- 如果同时传 `size` 与 `aspect_ratio`，两者必须对应相同方向和分辨率。
- 图片参考总数最多 4 个，统计范围包括 `image_url`、`image_urls`、`start_image_url`、`end_image_url`。
- 视频参考总数最多 3 个，统计范围包括 `video_url` 和 `video_reference`。
- 音频参考最多 1 个。
- 图片支持 PNG、JPEG、WebP、GIF、AVIF；视频支持 MP4；音频支持 MP3、WAV。
- 图片可传公网 HTTP/HTTPS URL 或受支持的图片 `data:` URL；视频和音频必须使用公网 HTTP/HTTPS URL。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 六个 Video 2.0 模型名之一 |
| `prompt` | string | 是 | 视频提示词，最长 5000 个字符 |
| `duration` | number | 否 | 4～15 秒 |
| `aspect_ratio` | string | 否 | `9:16`、`16:9`、`1:1` |
| `resolution` | string | 否 | 根据模型固定传 `720p` 或 `480p` |
| `size` | string | 否 | 与模型和画幅相符的像素尺寸 |
| `async` | boolean | 否 | 建议传 `true` |
| `image_url` | string | 否 | 单张参考图片 |
| `image_urls` | array | 否 | 多张参考图片 |
| `start_image_url` | string | 否 | 首帧图片 |
| `end_image_url` | string | 否 | 尾帧图片 |
| `video_url` | string | 否 | 单个参考视频 |
| `video_reference` | array | 否 | 多个参考视频，元素格式为 `{ "url": "..." }` |
| `audio_url` | string | 否 | 单个参考音频 |

## 14. Videos 4

### 14.1 `videos-4（4图3视频1音频）`

文生视频：

```bash
curl https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer $LINKSKY_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "videos-4（4图3视频1音频）",
    "prompt": "现代城市从白天过渡到夜晚的延时摄影，镜头稳定，灯光层次丰富",
    "duration": 8,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

多媒体参考：

```json
{
  "model": "videos-4（4图3视频1音频）",
  "prompt": "使用参考图片中的人物和产品，参考视频的运镜，并按照音频节奏生成商业广告",
  "duration": 10,
  "ratio": "16:9",
  "resolution": "720p",
  "referenceImages": [
    "https://example.com/person.png",
    "https://example.com/product.png",
    "https://example.com/scene.png",
    "https://example.com/logo.png"
  ],
  "referenceVideos": [
    "https://example.com/camera-1.mp4",
    "https://example.com/camera-2.mp4"
  ],
  "referenceAudios": [
    "https://example.com/music.mp3"
  ]
}
```

说明：

- `prompt` 不能为空，最长 5000 个字符。
- `duration` 支持 4～15 秒。
- `ratio` 支持 `16:9`、`9:16`、`1:1`。
- `resolution` 支持 `720p`、`480p`。
- 最多 4 张参考图片、3 个参考视频、1 个参考音频。
- 参考素材必须使用公网可访问的 HTTP/HTTPS URL。
- 不支持 `first_image` 和 `last_image`。

提交参数：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传 `videos-4（4图3视频1音频）` |
| `prompt` | string | 是 | 视频提示词，最长 5000 个字符 |
| `duration` | number | 否 | 4～15 秒 |
| `ratio` | string | 否 | `16:9`、`9:16`、`1:1` |
| `resolution` | string | 否 | `720p` 或 `480p` |
| `referenceImages` | array | 否 | 参考图片 URL，最多 4 张 |
| `referenceVideos` | array | 否 | 参考视频 URL，最多 3 个 |
| `referenceAudios` | array | 否 | 参考音频 URL，最多 1 个 |

## 15. 新增模型查询任务

本次新增的 11 个模型全部使用原文档相同的查询接口：

```bash
curl https://ghlink.top/v1/video/async-generations/task_xxx \
  -H "Authorization: Bearer $LINKSKY_API_KEY"
```

处理中、完成、失败响应格式以及结果 URL 读取顺序继续以原文档第 6 章为准，不新增第二套解析逻辑。

## 16. 新增模型常见参数错误

| 模型系列 | HTTP 状态码 | 常见消息或原因 |
| --- | --- | --- |
| MiniMax H3 | 400 | `prompt` 为空或过长、时长或画幅不支持、参考图片/音频数量或格式不符合要求 |
| SD2 / Videos 4 | 400 | `prompt is required`、`duration must be between 4 and 15`、`ratio must be 16:9, 9:16, or 1:1`、`resolution must be 720p or 480p`、参考素材数量超限 |
| Video 2.0 | 400 | `prompt is required`、`prompt must not exceed 5000 characters`、`duration must be between 4 and 15`、`aspect_ratio must be 9:16, 16:9, or 1:1`、`size conflicts with aspect_ratio`、参考素材数量或格式不符合要求 |
| 全部新增模型 | 401 | Token 无效、过期或缺失 |
| 全部新增模型 | 403 | Token 没有模型或分组访问权限 |
| 全部新增模型 | 404 | `task_id` 不存在 |
| 全部新增模型 | 429 | 额度不足或请求频率过高 |
| 全部新增模型 | 500/503 | 上游提交、轮询或可用渠道异常 |

## 17. 接入检查清单

- 请求地址使用 `https://ghlink.top`，不要重复拼接 `/v1`。
- `Authorization` 必须是 `Bearer` 加一个空格再加 API Key。
- 模型名必须从模型广场复制，不要自行修改大小写、括号或后缀。
- 固定分辨率的 `sd2-1080P(933按秒)` 和 `sd2-4k(933按秒)` 不传 `resolution`。
- Video 2.0 模型的 `resolution` 必须与模型名中的 720p/480p 档位一致。
- SD2 和 Videos 4 使用 `ratio`、`referenceImages`、`referenceVideos`、`referenceAudios`。
- MiniMax H3 和 Video 2.0 使用 `aspect_ratio` 以及各自的图片、视频、音频字段。
- 提交成功后保存 `task_id`，不要使用本地自增 ID 代替。
- 轮询间隔建议 3～5 秒，最长等待时间建议 5～10 分钟。
- 参考素材 URL 必须允许 GHLINK 和上游服务从公网访问。
