# GHLINK 最新 API 对接文档

更新日期：2026-09-20

本文档按当前网站代码整理，面向需要调用网站 API 的下游开发者。文档只描述对外调用方式，不暴露上游真实地址、渠道密钥、内部模型映射或后台实现细节。

示例 Base URL 使用 `https://ghlink.top`。私有部署请替换为你的网站域名。

## 1. 通用规则

所有 API 请求都使用 Bearer Token：

```http
Authorization: Bearer sk-你的令牌
```

JSON 请求请同时带上：

```http
Content-Type: application/json
```

客户侧不要在浏览器前端、公开仓库、App 安装包里暴露 API Token。推荐由自己的服务端读取环境变量后调用。

## 2. 公开接口总览

| 场景 | 推荐接口 |
| --- | --- |
| 查询可用模型 | `GET /v1/models` |
| 创作中心模型目录 | `GET /v1/creation/models?mode=image`、`GET /v1/creation/models?mode=video` |
| 对话补全 | `POST /v1/chat/completions` |
| Responses | `POST /v1/responses` |
| 图片生成 | `POST /v1/images/generations` |
| 图片编辑 | `POST /v1/images/edits` |
| 文本转语音 | `POST /v1/audio/speech` |
| 语音转文本 | `POST /v1/audio/transcriptions` |
| 翻译音频 | `POST /v1/audio/translations` |
| Embeddings | `POST /v1/embeddings` |
| 异步视频生成 | `POST /v1/video/async-generations` |
| 异步视频查询 | `GET /v1/video/async-generations/{task_id}` |
| OpenAI 兼容视频生成 | `POST /v1/videos` |
| OpenAI 兼容视频查询 | `GET /v1/videos/{task_id}` |
| 视频内容代理 | `GET /v1/videos/{task_id}/content` |
| 创作中心图片生成 | `POST /v1/creation/images/generations` |
| 创作中心视频生成 | `POST /v1/creation/video/async-generations` |
| 创作中心任务查询 | `GET /v1/creation/tasks/{task_id}` |

兼容接口仍可用：`POST /v1/video/generations`、`GET /v1/video/generations/{task_id}`、`POST /v1/videos/generations`、`GET /v1/videos/generations/{task_id}`、`POST /v1/videos/{video_id}/remix`。

## 3. 媒体结果 URL 规则

当前网站支持管理员配置多个图床/媒体存储。生成结果返回给客户前，系统会尽量把上游生成的图片、视频、音频转存到已配置的图床。

| 媒体类型 | 对外表现 |
| --- | --- |
| 图片 URL | 成功转存后，`data[].url` 返回图床 URL；转存失败保留上游 URL |
| 图片 base64 | 成功转存后补充/替换为 `data[].url`；转存失败时保留原 `b64_json`，并按旧逻辑保存本地预览 |
| 视频 URL | 任务成功后优先把结果 URL 转存到图床；转存失败保留上游 URL |
| 音频二进制 | 响应体仍然是音频二进制；转存成功时额外返回响应头 `X-Media-Storage-URL` |
| 流式音频 | 保持流式输出，不做转存 |

图床上传失败不会让已经成功的生成任务变成失败。客户侧应始终以业务结果 URL 或原始二进制响应为准。

## 4. 图片生成

### 4.1 创建图片

```bash
curl -X POST "https://ghlink.top/v1/images/generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image2",
    "prompt": "一张赛博风格的猫咪海报，霓虹灯，高清细节",
    "size": "1024x1024",
    "n": 1,
    "response_format": "url"
  }'
```

### 4.2 图片响应

```json
{
  "created": 1780213835,
  "data": [
    {
      "url": "https://media.example.com/generated/image.png",
      "b64_json": "",
      "revised_prompt": ""
    }
  ]
}
```

说明：

- `data[].url` 可能是图床 URL，也可能是上游 URL，取决于管理员是否配置图床以及上传是否成功。
- 如果模型返回 `b64_json`，且图床转存成功，客户端优先使用 `url`。
- 图片返回中可能包含 `metadata`，客户侧不应依赖未声明字段。

## 5. 视频生成

视频为异步任务。客户端先提交任务并保存 `task_id`，随后轮询查询接口。建议普通模型每 10-20 秒查询一次；对有明确慢轮询要求的线路，建议不低于 20 秒。

### 5.1 推荐异步接口

```bash
curl -X POST "https://ghlink.top/v1/video/async-generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "video-2.5",
    "prompt": "清晨海边公路，跑车沿海岸线行驶，电影级航拍，光影自然",
    "duration": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'
```

提交成功示例：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-2.5",
  "status": "queued",
  "progress": 0,
  "created_at": 1780213835,
  "seconds": "8"
}
```

查询任务：

```bash
curl -X GET \
  "https://ghlink.top/v1/video/async-generations/task_xxxxxxxxxxxxxxxx" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

完成示例：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "video-2.5",
  "status": "completed",
  "progress": 100,
  "created_at": 1780213835,
  "completed_at": 1780213958,
  "seconds": "8",
  "metadata": {
    "url": "https://media.example.com/generated/video.mp4"
  }
}
```

客户侧读取视频结果时，优先使用 `metadata.url`。如果需要通过本站代理下载，也可以请求：

```http
GET /v1/videos/{task_id}/content
```

### 5.2 OpenAI 兼容视频接口

```bash
curl -X POST "https://ghlink.top/v1/videos" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -F "model=sora-2" \
  -F "prompt=一只小猫在木地板上跳舞，镜头轻微推近" \
  -F "seconds=8"
```

查询：

```http
GET /v1/videos/{task_id}
```

返回结构与第 5.1 节相同，完成后仍优先读取 `metadata.url`。

### 5.3 创作中心 Token 接口

创作中心接口适合只需要媒体模型目录和媒体生成任务的服务端客户。

查询模型：

```bash
curl "https://ghlink.top/v1/creation/models?mode=video" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

提交视频：

```http
POST /v1/creation/video/async-generations
```

查询任务：

```http
GET /v1/creation/tasks/{task_id}
```

任务查询响应示例：

```json
{
  "success": true,
  "message": "",
  "data": {
    "task_id": "task_xxxxxxxxxxxxxxxx",
    "request_id": "",
    "status": "succeeded",
    "progress": 100,
    "result_url": "https://media.example.com/generated/video.mp4",
    "result_type": "video",
    "actual_quota": 1000,
    "billing_status": "settled",
    "created_at": 1780213835,
    "updated_at": 1780213958
  }
}
```

## 6. 视频请求字段

所有视频任务至少需要：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 模型 ID，必须按模型目录返回值原样传递 |
| `prompt` | string | 是 | 视频提示词 |
| `duration` | integer | 否 | 时长，单位秒；部分接口也兼容 `seconds` |
| `seconds` | string/number | 否 | OpenAI 兼容视频常用时长字段 |
| `ratio` | string | 否 | 宽高比；部分接口也兼容 `aspect_ratio` |
| `aspect_ratio` | string | 否 | 宽高比别名，会映射到 `ratio` |
| `resolution` | string | 否 | 例如 `720p`、`480p` |
| `size` | string | 否 | OpenAI/Sora 风格尺寸，例如 `720x1280` |
| `image_url` | string | 否 | 单张参考图片 |
| `image_urls` | string[] | 否 | 多张参考图片 |
| `images` | string[] | 否 | 多张参考图片 |
| `start_image_url` | string | 否 | 首帧图片 URL，具体支持情况看模型 |
| `end_image_url` | string | 否 | 尾帧图片 URL，具体支持情况看模型 |
| `video_url` | string | 否 | 单个参考视频 |
| `video_reference` | object[] | 否 | 参考视频对象数组，常用字段为 `url` |
| `audio_url` | string | 否 | 单个参考音频 |
| `audio_reference` | object[] | 否 | 参考音频对象数组，常用字段为 `url` |
| `referenceImages` | string[] | 否 | 参考图片数组 |
| `referenceVideos` | string[] | 否 | 参考视频数组 |
| `referenceAudios` | string[] | 否 | 参考音频数组 |

不要传空字符串。公网素材必须是可由服务端直接访问的 `http://` 或 `https://` URL，不能依赖 Cookie、登录态、内网地址或短期会失效的网页链接。

### 6.1 Video 2.0 / Video 2.5

适用模型：

- `video-2.0`、`video-2.0-fast`、`video-2.0-mini`
- `video-2.0-480p`、`video-2.0-fast-480p`、`video-2.0-mini-480p`
- `video-2.5`、`video-2.5-480p`

字段限制：

| 模型 | 时长 | 分辨率 | 图片 | 视频 | 音频 |
| --- | --- | --- | --- | --- | --- |
| `video-2.0*` | 4-15 秒 | 非 `-480p` 为 `720p`，`-480p` 为 `480p` | 最多 4 | 最多 3 | 最多 1 |
| `video-2.5*` | 4-30 秒 | 非 `-480p` 为 `720p`，`-480p` 为 `480p` | 最多 30 | 最多 10 | 最多 10 |

支持比例：`9:16`、`16:9`、`1:1`。

参考素材格式限制：

- 图片：PNG、JPEG、WebP、GIF、AVIF；也支持合法的图片 data URL。
- 视频：MP4。
- 音频：MP3、WAV。

### 6.2 Seedance 2.0 / 2.5

适用模型包括 `seedance-2.0`、`seedance-2.5`、`sd-2.0-933`、`sd-2-c8` 以及模型目录中映射到 Seedance 2.x 的公开名称。

Seedance 2.0：

| 字段 | 限制 |
| --- | --- |
| `prompt` | 最长 5000 字符 |
| `duration` | 4-15 秒 |
| `ratio` | `1:1`、`16:9`、`9:16`、`4:3`、`3:4` |
| `resolution` | 仅 `720p` |
| 图片参考 | `start_image_url`、`end_image_url`、`referenceImages` 合计最多 9 |
| 视频参考 | `referenceVideos` 最多 3 |
| 音频参考 | `referenceAudios` 最多 3 |
| 全部参考素材 | 最多 15 |

Seedance 2.5：

| 字段 | 限制 |
| --- | --- |
| `prompt` | 最长 5000 字符 |
| `duration` | 4-29 秒 |
| `ratio` | `16:9`、`9:16`、`1:1` |
| `resolution` | `720p`、`480p` |
| 图片参考 | 最多 30 |
| 视频参考 | 最多 10 |
| 音频参考 | 最多 10 |
| 全部参考素材 | 最多 50 |

Seedance 2.x 不支持 `first_image` 和 `last_image`。首尾帧请使用 `start_image_url`、`end_image_url` 或按模型目录说明放入参考图数组。

### 6.3 `/v1/videos` 系列

适用模型包括 `videos-standard`、`videos-fast`、`videos-mini`、`videos-4`、`videos-4-fast`、`videos-4-mini`，以及模型名规范化后以 `sd2` 开头的模型。

通用限制：

- `prompt` 最长 5000 字符。
- `duration` 支持 4-15 秒。
- `ratio` 支持 `16:9`、`9:16`、`1:1`。
- `resolution` 支持 `720p`、`480p`。
- 不支持 `first_image` 和 `last_image`。

素材数量：

| 模型 | 图片 | 视频 | 音频 |
| --- | --- | --- | --- |
| `videos-4*` | 最多 4 | 最多 3 | 最多 1 |
| 其他 `/v1/videos` 系列 | 最多 9 | 最多 3 | 最多 3 |

### 6.4 Sora 兼容模型

适用模型包括 `sora2`、`sora-2`、`sora-2-pro`。

常用字段：

| 字段 | 说明 |
| --- | --- |
| `prompt` | 视频提示词 |
| `seconds` | 秒数，常用 `4`、`8`、`12` 等 |
| `size` | `sora-2` 支持 `720x1280`、`1280x720`；`sora-2-pro` 额外支持 `1792x1024`、`1024x1792` |
| `input_reference` | 参考图片文件，multipart 场景使用 |

### 6.5 004 / 全能 / 官转 / 特价系列

当前代码已识别以下模型族：

- `004系列/...`
- `omni-video-1`、`omni-video-1-fast`、`omni-video-1-pro`
- `sd2.0`、`sd-mini`、`sd-2.5`
- `wan3.0-video`、`wan3.0-video-prime`、`wan3.0-image`、`wan3.0-image-prime`
- 以 `官转` 开头的模型名
- `grok-imagine-video-1.5`

004 系列通用限制：

- `prompt` 最长 2000 字符。
- `ratio` 支持 `16:9`、`9:16`、`1:1`、`4:3`、`3:4`。
- `resolution` 支持 `480p`、`720p`、`768p`、`2k`。
- 模型名包含 `sd2.5` 时，时长一般为 4-30 秒。
- 模型名包含 `minimax` 或 `sd2.0` 时，时长一般为 5-15 秒。
- 模型名包含 `video-editing` 时，必须正好传 1 个视频和 1 张图片，且时长不超过 30 秒。
- 模型名包含 `8图3音频` 时，最多 8 图、0 视频、3 音频。
- 模型名包含 `原生过人脸9图` 时，最多 9 图、0 视频、0 音频。
- 模型名包含 `30图4-30秒` 时，最多 30 图、0 视频、0 音频。
- 模型名包含 `10-10-10` 时，最多 10 图、10 视频、10 音频。
- 模型名包含 `30-10-10` 时，最多 30 图、10 视频、10 音频。
- 模型名包含 `grok` 时，最多 7 图、0 视频、0 音频。

Omni 模型限制：

- `prompt` 最长 5000 字符。
- `duration` 只能是 `3`、`5`、`8`。
- `aspect_ratio` 支持 `16:9`、`9:16`、`1:1`。
- 可传 `image_url` 作为参考图。

## 7. 音频接口

### 7.1 文本转语音

```bash
curl -X POST "https://ghlink.top/v1/audio/speech" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -o speech.mp3 \
  -D headers.txt \
  -d '{
    "model": "tts-1",
    "voice": "alloy",
    "input": "你好，这是一次语音生成测试。",
    "response_format": "mp3"
  }'
```

响应体是音频二进制。若管理员已配置图床且上传成功，响应头会包含：

```http
X-Media-Storage-URL: https://media.example.com/generated/audio.mp3
```

上传失败时不会影响音频响应体。

### 7.2 语音转文本 / 翻译

```http
POST /v1/audio/transcriptions
POST /v1/audio/translations
```

这两个接口按 OpenAI 兼容格式提交音频文件。它们不是生成媒体结果的接口，不会产生图床 URL。

## 8. 错误处理

标准 relay 接口通常返回 OpenAI 风格错误：

```json
{
  "error": {
    "message": "错误说明",
    "type": "new_api_error",
    "param": "",
    "code": "invalid_request"
  }
}
```

视频任务提交阶段也可能返回任务错误：

```json
{
  "code": "invalid_request",
  "message": "prompt is required",
  "data": {
    "category": "validation",
    "retryable": false
  }
}
```

视频任务轮询到失败状态时，响应中会带 `error`：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "status": "failed",
  "progress": 100,
  "error": {
    "message": "生成失败，请检查参考素材 URL 是否可访问",
    "code": "provider_failed"
  }
}
```

客户侧建议：

- `400`：检查请求字段、模型名、素材 URL、素材数量和格式。
- `401/403`：检查 API Token、账户权限和余额。
- `429`：降低并发或放慢重试。
- `5xx`：可以指数退避重试；不要立即高频重放。
- 异步任务失败后不要无限轮询，应展示 `error.message` 并让用户重新提交。

## 9. 管理员图床配置

后台入口：系统设置 -> 运维 -> Media storage。

当前默认新增项适配 GHLINK ImgHub：

| 配置项 | 值 |
| --- | --- |
| Upload URL | `https://media.ghlink.top/upload` |
| Auth header | `authCode` |
| Auth prefix | 留空 |
| File field | `file` |
| Response URL path | `url` |
| Token | 管理员填写图床 Token |

支持多个图床配置。系统会按 `priority` 从小到大依次尝试，某个图床失败后继续尝试下一个；全部失败时保留上游 URL 或原音频响应。

图床成功响应必须是 JSON，并且默认能解析出顶层 `url` 字段：

```json
{
  "url": "https://media.example.com/path/to/file.png"
}
```

如果图床使用文档化的 JSON 包装结构，例如 `{"data":{"url":"https://..."}}`，管理员可以把 `Response URL path` 设置为 `data.url`。系统只读取管理员明确配置的点号分隔字段路径，不会猜测未记录的字段。

如果图床只返回纯文本 `Saved`、JSON 字符串 `"Saved"` 或只有 `{"status":"Saved"}`，但没有 URL 字段，当前网站无法知道最终文件地址，会按上传失败回退并保留上游 URL。

设置页提供“测试上传”按钮。测试会向已保存的 provider 上传一个很小的 PNG 文件，并验证 HTTP 状态、JSON URL 字段和最终 URL 格式；测试不会改变生成结果。新增测试接口为 root 专用：

```http
POST /api/option/media_storage/test
Content-Type: application/json

{"provider_id":"ghlink-imghub"}
```

测试成功响应：

```json
{
  "success": true,
  "data": {
    "provider_id": "ghlink-imghub",
    "url": "https://media.example.com/path/to/test.png"
  }
}
```

请先保存 provider，再执行测试；测试接口读取数据库中已保存的配置，Token 不会返回给前端。

管理员 API 仅 root 可用：

```http
GET /api/option/media_storage
PUT /api/option/media_storage
```

配置结构：

```json
{
  "providers": [
    {
      "id": "ghlink-imghub",
      "name": "GHLINK ImgHub",
      "enabled": true,
      "upload_url": "https://media.ghlink.top/upload",
      "auth_header": "authCode",
      "auth_prefix": "",
      "token": "你的图床 Token",
      "field_name": "file",
      "priority": 0,
      "response_url_path": "url"
    }
  ]
}
```

读取配置时，已有 token 会显示为 `********`；保存时保留 `********` 可继续使用原 token。

## 10. 实现核对表

| 文档定义 | 当前代码调用位置 | 当前实现 |
| --- | --- | --- |
| `POST /v1/images/generations` | `router/relay-router.go` -> `controller.Relay` -> `relay/image_handler.go` | OpenAI 图片兼容；响应会经过图片结果捕获/图床转存 |
| `POST /v1/audio/speech` | `router/relay-router.go` -> `controller.Relay` -> `relay/audio_handler.go` | 返回音频二进制；非流式成功时可附加 `X-Media-Storage-URL` |
| `POST /v1/video/async-generations` | `router/video-router.go` -> `controller.RelayTask` -> `relay/relay_task.go` | 异步视频任务提交，返回公开 `task_id` |
| `GET /v1/video/async-generations/{task_id}` | `router/video-router.go` -> `controller.RelayTaskFetch` -> `relay/relay_task.go` | 查询公开视频任务状态，成功时返回 `metadata.url` |
| `POST /v1/videos` | `router/video-router.go` -> `controller.RelayTask` -> Sora task adaptor | OpenAI 兼容视频任务提交 |
| `GET /v1/videos/{task_id}` | `router/video-router.go` -> `controller.RelayTaskFetch` -> Sora task adaptor | OpenAI 兼容视频任务查询 |
| `GET /v1/videos/{task_id}/content` | `router/video-router.go` -> `controller.VideoProxy` | 视频内容代理，支持 token 或登录态 |
| `GET /v1/creation/models` | `router/relay-router.go` -> `controller.GetCreationTokenModels` | Token 客户查询图片/视频模型目录 |
| `POST /v1/creation/video/async-generations` | `router/relay-router.go` -> `controller.CreationTokenRelayTask` | 创作中心 token 视频任务提交 |
| `GET /v1/creation/tasks/{task_id}` | `router/relay-router.go` -> `controller.GetCreationTask` | 创作中心任务查询，返回安全字段 |
| `POST /api/option/media_storage/test` | `router/api-router.go` -> `controller.TestMediaStorage` -> `service.TestMediaStorageProvider` | root 测试已保存 provider，验证 multipart、认证和响应 URL |
| 图床配置 | `router/api-router.go` -> `controller/media_storage.go` -> `service/media_storage.go` | root 后台配置，多 provider，失败回退；响应 URL 路径显式配置 |

本文档没有覆盖未实现接口，例如 `/v1/files`、`/v1/fine-tunes`、`/v1/images/variations`。这些路由当前会返回未实现。
