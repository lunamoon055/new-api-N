# 13 个视频模型下游接入文档

本文描述当前网关已接入的 13 个视频模型对**下游 API 客户端**开放的调用方式。请求发送到你所使用的本网关域名，不要把供应商的 API Key 配到下游；下游只使用本网关签发的 API Key。模型 ID、参数限制以本文对应的当前接入代码为准，模型可用性与价格以部署实例的实时配置为准。

这是一份接口与能力文档，不表示 13 个模型已逐个完成真实付费生成验收。上线前请按实际渠道、Key 权限和定价配置进行端到端验证。

## 1. 接入地址和鉴权

将 `API_BASE_URL` 换成你的网关地址，不要使用供应商文档的域名：

```bash
export API_BASE_URL="https://your-gateway.example.com"
export API_KEY="sk-your-gateway-token"
```

所有调用都在服务端发送 `Authorization: Bearer $API_KEY`；提交任务还需 `Content-Type: application/json`。API Key 不应放进网页、移动端或公开仓库。

可先查询当前 Key 可见的模型：

```bash
curl -sS "$API_BASE_URL/v1/models" \
  -H "Authorization: Bearer $API_KEY"
```

模型必须对该 Key 的分组开放，且管理员已配置对应渠道和**该精确模型 ID**的定价，才能提交任务。以下名称中的中文、大小写、连字符都是模型 ID 的一部分，不能使用模型卡片的简写，也不能将近似名称合并。

## 2. 模型清单与参数上限

所有模型统一走 `POST /v1/videos`，为异步任务。`prompt` 必填。表中“图 / 视频 / 音频”是**最多允许的参考素材 URL 数量**，`0` 表示不支持；“至少 1 图”表示图生视频必需。时长单位为秒，列出的区间包含两端。

| # | 精确模型 ID | `duration` | `resolution` | 比例组 | 图 / 视频 / 音频 | 特别要求 |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `特价dola2.5-900-720P-固定30s` | 固定 `30` | `720p` | 宽 | `1–9 / 0 / 0` | 至少 1 图 |
| 2 | `特价dola-sd2.0-900-720p` | `5`、`10`、`15` | `720p` | 宽 | `1 / 0 / 0` | 恰好 1 图 |
| 3 | `TTP-grok` | `6–15` | `480p` 或 `720p` | 标准 | `0–7 / 0 / 0` | 必须显式传 `resolution` |
| 4 | `gz稳定sd2.0-903-720P` | `5–15` | `720p` | 宽 | `0–9 / 0 / 0–3` | 可使用参考音频 |
| 5 | `Xminimex-h3-2K-933` | `5–15` | `2k` | 宽 | `0–9 / 0–3 / 0–3` | 支持多模态参考 |
| 6 | `特价sd2.0-720p-933-过脸` | `5–15` | `720p` | 宽 | `0–9 / 0–3 / 0–3` | 与第 12 个模型分别定价 |
| 7 | `特价稳sd-2.5-30s-720P可10图` | 固定 `30` | `720p` | 宽 | `0–10 / 0 / 0` | 图片使用 `reference_images` |
| 8 | `特价稳sd2.0-933-720p` | `5–15` | `720P` | 宽 | `0–9 / 0–3 / 0–3` | 以当前接入的 `5–15` 秒范围为准 |
| 9 | `gz稳定sd2.5-720全参过脸` | `4–30` | `720P` | 宽 | `0–30 / 0–10 / 0–10` | 支持多模态参考 |
| 10 | `gz稳定sd2.5-480全参过脸` | `4–30` | `480P` | 宽 | `0–30 / 0–10 / 0–10` | 支持多模态参考 |
| 11 | `gz稳定sd2.5-1080全参过脸` | `4–30` | `1080p` | 宽 | `0–30 / 0–10 / 0–10` | 支持多模态参考 |
| 12 | `特价sd2.0-933-过脸` | `4–15` | `720p` | 四种 | `0–9 / 0–3 / 0–3` | 传音频时还须有图或视频；与第 6 个模型分别定价 |
| 13 | `MiniMaxH3` | `4–15` | `2K` | 标准 | `0–9 / 0 / 0–3` | 支持首尾帧；传音频时还须有图片或首尾帧 |

比例组：

- **宽**：`16:9`、`9:16`、`1:1`、`4:3`、`3:4`、`21:9`。
- **标准**：`16:9`、`9:16`、`1:1`。
- **四种**：`16:9`、`9:16`、`1:1`、`4:3`。

`ratio` 省略时默认 `16:9`。固定分辨率的模型可省略 `resolution`，网关会补齐表中值；若主动传入，必须与表中分辨率一致（匹配时不区分大小写）。两个固定 30 秒的模型可省略 `duration`，网关会补 `30`。其余模型必须传入整数秒数。第 8 个模型的供应商描述与参数表对最短时长有差异，本接入采用较窄的 `5–15` 秒范围。

## 3. 创建视频任务

```http
POST /v1/videos
Authorization: Bearer <API_KEY>
Content-Type: application/json
```

通用字段：

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `model` | string | 表中的精确模型 ID，必填 |
| `prompt` | string | 非空视频提示词，必填 |
| `duration` | integer | 生成秒数；固定 30 秒模型可省略 |
| `ratio` | string | 画面比例，省略时为 `16:9` |
| `resolution` | string | 第 3 个模型必填；其他模型可省略 |
| `referenceImages` | string[] | 参考图片的公网 URL；网关会映射到模型需要的字段 |
| `referenceVideos` | string[] | 参考视频的公网 URL，仅在表中上限大于 0 时可用 |
| `referenceAudios` | string[] | 参考音频的公网 URL，仅在表中上限大于 0 时可用 |
| `start_image_url` / `end_image_url` | string | 仅 `MiniMaxH3` 支持的首帧 / 尾帧图片 URL；计入最多 9 张图 |
| `generate_audio` | boolean | 可选；如果提供，必须是 JSON 布尔值 |

下游可以统一使用 `referenceImages`、`referenceVideos`、`referenceAudios`；网关会按模型转换为供应商要求的 `images`、`reference_images`、`image_urls` 等字段。也可直接传这些模型专用字段，但同一批素材不要同时用多种别名重复提交。素材地址需为网关及供应商可访问的公网 HTTP(S) URL，不可依赖登录 Cookie，也不能在 URL 中嵌入用户名或密码。**不要传 `size`**；请使用 `resolution`。

### 3.1 文生视频示例

```bash
curl -sS -X POST "$API_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "特价sd2.0-720p-933-过脸",
    "prompt": "电影感街景，人物向镜头走来，动作流畅",
    "duration": 15,
    "ratio": "16:9"
  }'
```

### 3.2 必须带图片的模型

```bash
curl -sS -X POST "$API_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "特价dola2.5-900-720P-固定30s",
    "prompt": "参考图中的人物在海边回头，稳定镜头",
    "duration": 30,
    "ratio": "16:9",
    "referenceImages": ["https://cdn.example.com/reference.jpg"]
  }'
```

### 3.3 可选分辨率、多模态与首尾帧

`TTP-grok` 必须给出 `resolution`，例如 `"resolution": "480p"`。第 9–11 个模型可组合参考图、视频和音频，并遵守各自的数量上限，例如：

```json
{
  "model": "gz稳定sd2.5-720全参过脸",
  "prompt": "参考人物与镜头运动，生成自然连贯的视频",
  "duration": 10,
  "ratio": "16:9",
  "referenceImages": ["https://cdn.example.com/face.jpg"],
  "referenceVideos": ["https://cdn.example.com/motion.mp4"],
  "referenceAudios": ["https://cdn.example.com/music.mp3"]
}
```

`MiniMaxH3` 的首尾帧写法：

```json
{
  "model": "MiniMaxH3",
  "prompt": "从首帧平滑过渡到尾帧",
  "duration": 8,
  "start_image_url": "https://cdn.example.com/start.jpg",
  "end_image_url": "https://cdn.example.com/end.jpg"
}
```

`MiniMaxH3` 不支持参考视频；它的普通参考图和首尾帧合计最多 9 张。若使用音频参考，必须同时提供图片或首尾帧。第 12 个模型若使用音频参考，必须同时提供图片或视频参考。

## 4. 异步任务查询与视频下载

创建接口返回的 `id`（兼容字段 `task_id`）是**本网关的公开任务 ID**，不是供应商内部任务号。保存这个 ID，后续使用同一网关和同一用户的 API Key 查询。创建接口返回 HTTP 200 且状态为 `queued` 只代表任务已受理，不代表视频已生成。

创建响应示例（具体可选字段可能不同）：

```json
{
  "id": "task_example",
  "task_id": "task_example",
  "object": "video",
  "model": "特价sd2.0-720p-933-过脸",
  "status": "queued",
  "progress": 0
}
```

每隔约 5–10 秒查询，保持原任务 ID；长视频可能需要数分钟，不要因为一次查询暂时未完成就重新提交：

```bash
export TASK_ID="task_example"
curl -sS "$API_BASE_URL/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $API_KEY"
```

| 查询响应 `status` | 下游处理 |
| --- | --- |
| `queued`、`in_progress` | 继续轮询，不要新建任务 |
| `unknown` | 暂按未完成处理，继续轮询；供应商刚受理时可能短暂返回未知状态 |
| `completed` | 读取 `metadata.url` 或调用视频内容接口 |
| `failed` | 停止轮询，读取 `error.message` 并按业务需要处理 |

完成响应示例：

```json
{
  "id": "task_example",
  "task_id": "task_example",
  "object": "video",
  "model": "特价sd2.0-720p-933-过脸",
  "status": "completed",
  "progress": 100,
  "metadata": {
    "url": "https://cdn.example.com/result.mp4"
  }
}
```

`metadata.url` 可能是结果文件 URL，也可能是本网关的视频代理地址。希望用统一鉴权方式下载时，可在任务 `completed` 后调用：

```bash
curl -fL "$API_BASE_URL/v1/videos/$TASK_ID/content" \
  -H "Authorization: Bearer $API_KEY" \
  -o video.mp4
```

内容接口只允许任务所属用户访问；任务未完成时不能下载。如果完成后仍无法下载，应保留任务 ID 和查询响应供排查，不要用供应商内部任务号自行拼接 URL。

## 5. 错误、重试与计费

- 参数不符合该模型限制时，提交会返回 4xx；先按错误消息修正时长、分辨率、比例或素材数量。
- 鉴权失败时，检查的是**本网关下游 API Key**和模型/分组权限；供应商的 Key 由渠道管理员在网关侧配置。
- 创建请求超时或网络中断时，先查本地业务记录是否已经保存任务 ID，避免重复创建并重复计费。不要假定 `/v1/videos` 的 `Idempotency-Key` 一定提供本地幂等保障。
- 查询时遇到临时网络错误或 5xx，请保留原任务 ID 并退避重试；只有任务查询明确返回 `failed` 才按任务失败处理。
- 价格由本网关管理员针对**精确模型 ID**配置。第 6 个 `特价sd2.0-720p-933-过脸` 与第 12 个 `特价sd2.0-933-过脸` 是两个模型，价格不互用。未配置价格的模型可能被网关拒绝提交。不要将某个模型的截图价格当成 13 个模型的统一报价。

供应商原始接口文档供核对模型能力：[小橘子 API 文档](https://api.xjzapi.cc/docs)。下游实际调用以上本网关接口；供应商文档中的 Base URL、API Key 和内部任务号不用于本接入。
