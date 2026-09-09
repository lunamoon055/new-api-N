# GHLINK Seedance 2.0 视频生成 API

本文档面向需要通过 GHLINK 接入 Seedance 2.0 视频生成能力的开发者。

接口采用异步任务模式：客户端先提交生成任务并取得 `task_id`，随后定时查询任务状态；任务完成后，从响应中取得视频地址并及时下载保存。

## 1. 基础信息

| 项目 | 内容 |
| --- | --- |
| Base URL | `https://ghlink.top` |
| 认证方式 | `Authorization: Bearer sk-你的令牌` |
| 请求格式 | `application/json` |
| 模型名称 | `seedance2.0-c` |
| 创建任务 | `POST /v1/video/async-generations` |
| 查询任务 | `GET /v1/video/async-generations/{task_id}` |
| 建议轮询间隔 | 不低于 20 秒 |
| 结果有效期 | 约 10 小时，请及时下载保存 |

> 模型名称必须按模型广场显示的内容完整传递，包括中文括号、线路前缀、大小写和连字符。不要把平台内部模型名称作为下游请求参数。

## 2. 快速开始

### 2.1 创建文生视频任务

```bash
curl -X POST "https://ghlink.top/v1/video/async-generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seedance2.0-c",
    "prompt": "一只橘猫在阳光下的草地上奔跑，电影级光影，镜头平稳跟拍",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

提交成功后，请保存响应中的 `task_id`：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "",
  "model": "Seedance2.0-c",
  "status": "RUNNING",
  "progress": 0,
  "created_at": 1780213835
}
```

部分字段可能为空或为 `0`。是否成功受理应以 HTTP 状态码、`task_id` 和 `status` 为准。

### 2.2 查询任务状态

首次查询建议在提交任务 20 秒后进行，后续两次查询之间也应至少间隔 20 秒。

```bash
curl -X GET \
  "https://ghlink.top/v1/video/async-generations/task_xxxxxxxxxxxxxxxx" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

任务完成后，从 `metadata.url` 获取视频地址：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "seedance2.0-c",
  "status": "completed",
  "progress": 100,
  "created_at": 1780213835,
  "completed_at": 1780213958,
  "seconds": "10",
  "metadata": {
    "url": "https://ghlink.top/v1/videos/task_xxxxxxxxxxxxxxxx/content"
  }
}
```

## 3. 认证

所有请求都必须携带 API Token：

```http
Authorization: Bearer sk-你的令牌
```

创建任务时还需要指定 JSON 请求格式：

```http
Content-Type: application/json
```

请勿在浏览器前端、公开仓库或客户端安装包中暴露 API Token。推荐由服务端调用本接口，并通过环境变量读取密钥。

## 4. 调用流程

1. 准备提示词以及可选的图片、视频、音频公网地址。
2. 调用 `POST /v1/video/async-generations` 创建任务。
3. 保存响应中的 `task_id`。
4. 等待至少 20 秒后调用查询接口。
5. 当状态为 `queued`、`in_progress`、`PENDING` 或 `RUNNING` 时继续等待。
6. 当状态为 `completed` 或 `SUCCEEDED` 时读取 `metadata.url`。
7. 当状态为 `failed` 或以 `FAILED` 开头时停止轮询并记录错误。
8. 视频结果只临时保存约 10 小时，请及时下载到自己的对象存储。

## 5. 创建视频任务

### 5.1 接口

```http
POST /v1/video/async-generations
```

### 5.2 请求字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | 固定传模型广场中的公开名称 `seedance2.0-c` |
| `prompt` | string | 是 | 视频内容描述，不能为空；建议不超过 5000 个字符 |
| `duration` | integer | 建议 | 输出视频时长，单位为秒，支持 `5`～`15` |
| `ratio` | string | 建议 | 输出宽高比：`1:1`、`16:9`、`9:16` |
| `resolution` | string | 建议 | 当前模型固定使用 `720p` |
| `start_image_url` | string | 否 | 首帧图片的公网 HTTP/HTTPS 地址 |
| `end_image_url` | string | 否 | 尾帧图片的公网 HTTP/HTTPS 地址 |
| `referenceImages` | string[] | 否 | 参考图片地址列表，可作为人物、角色、风格或场景参考 |
| `referenceVideos` | string[] | 否 | 参考视频地址列表，可用于动作、表情、运镜或节奏参考 |
| `referenceAudios` | string[] | 否 | 参考音频地址列表，可用于声音、音色或节奏参考 |

没有参考素材时不传对应字段，不要传空字符串。所有素材地址必须能够从公网直接访问，不能依赖登录态、Cookie、临时网页或内网环境。

## 6. 生成方式示例

### 6.1 文生视频

```json
{
  "model": "seedance2.0-c",
  "prompt": "小猫在客厅里追逐一只红色毛线球，镜头缓慢推进，动作自然",
  "duration": 10,
  "ratio": "16:9",
  "resolution": "720p"
}
```

### 6.2 首帧图生视频

```json
{
  "model": "seedance2.0-c",
  "prompt": "画面中的人物自然转身看向镜头，头发随微风轻轻摆动",
  "duration": 10,
  "ratio": "9:16",
  "resolution": "720p",
  "start_image_url": "https://example.com/first-frame.png"
}
```

### 6.3 首尾帧生视频

```json
{
  "model": "seedance2.0-c",
  "prompt": "人物从首帧的站立姿态自然过渡到尾帧的奔跑姿态，保持人物身份和服装一致",
  "duration": 10,
  "ratio": "16:9",
  "resolution": "720p",
  "start_image_url": "https://example.com/first-frame.png",
  "end_image_url": "https://example.com/last-frame.png"
}
```

### 6.4 多图片参考

图片在数组中的顺序对应提示词中的“图1”“图2”等称呼。

```json
{
  "model": "seedance2.0-c",
  "prompt": "让图1中的人物穿上图2中的服装，在图3的街道场景中向前行走",
  "duration": 10,
  "ratio": "9:16",
  "resolution": "720p",
  "referenceImages": [
    "https://example.com/character.png",
    "https://example.com/clothing.png",
    "https://example.com/street.png"
  ]
}
```

### 6.5 图片、音频与视频综合参考

```json
{
  "model": "seedance2.0-c",
  "prompt": "图1中的歌手使用音频1的音色演唱，参考视频1的舞台动作和镜头节奏",
  "duration": 10,
  "ratio": "16:9",
  "resolution": "720p",
  "referenceImages": [
    "https://example.com/singer.png"
  ],
  "referenceVideos": [
    "https://example.com/stage-motion.mp4"
  ],
  "referenceAudios": [
    "https://example.com/voice.mp3"
  ]
}
```

## 7. 素材规则与限制

| 类型 | 数量限制 | 说明 |
| --- | --- | --- |
| 图片 | 最多 9 张 | 首帧、尾帧和 `referenceImages` 合并计数 |
| 视频 | 最多 3 个 | 通过 `referenceVideos` 提交 |
| 音频 | 最多 3 个 | 通过 `referenceAudios` 提交 |
| 全部参考素材 | 最多 15 个 | 图片、视频和音频合计 |

另外需要遵守以下规则：

- 图片尺寸不得低于 `300 × 300`。
- 素材地址必须以 `http://` 或 `https://` 开头，并允许服务端直接下载。
- 输入参考视频时长与输出视频时长之和不得超过 25 秒。
- 建议使用稳定的对象存储或 CDN 地址，并保证任务执行期间链接不会过期。
- 不要使用 `first_image` 或 `last_image`；首尾帧请分别使用 `start_image_url` 和 `end_image_url`。
- 提示词中可以使用“图1”“图2”“视频1”“音频1”等称呼引用对应数组中的素材。

## 8. 创建任务响应

创建接口是异步接口。成功受理时通常返回一个公开 `task_id`：

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "",
  "model": "seedance2.0-c",
  "status": "RUNNING",
  "progress": 0,
  "created_at": 1780213835
}
```

| 字段 | 类型 | 说明 |
| --- | --- | --- |
| `id` | string | 任务 ID，通常与 `task_id` 相同 |
| `task_id` | string | 后续查询任务时使用的公开任务 ID |
| `object` | string | 对象类型；提交阶段可能为空 |
| `model` | string | 请求使用的公开模型名称；部分提交响应中可能为空 |
| `status` | string | 当前任务状态 |
| `progress` | integer | 任务进度，范围通常为 0～100 |
| `created_at` | integer | Unix 时间戳；部分提交响应中可能为 0 |

客户端应完整保存 `task_id`，不要使用数据库自增 ID、请求 ID 或其他字段代替。

## 9. 查询任务

### 9.1 接口

```http
GET /v1/video/async-generations/{task_id}
```

`task_id` 必须使用创建任务接口返回的值。

### 9.2 处理中

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "seedance2.0-c",
  "status": "in_progress",
  "progress": 35,
  "created_at": 1780213835
}
```

### 9.3 已完成

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "seedance2.0-c",
  "status": "completed",
  "progress": 100,
  "created_at": 1780213835,
  "completed_at": 1780213958,
  "seconds": "10",
  "metadata": {
    "url": "https://ghlink.top/v1/videos/task_xxxxxxxxxxxxxxxx/content"
  }
}
```

### 9.4 已失败

```json
{
  "id": "task_xxxxxxxxxxxxxxxx",
  "task_id": "task_xxxxxxxxxxxxxxxx",
  "object": "video",
  "model": "seedance2.0-c",
  "status": "failed",
  "progress": 100,
  "created_at": 1780213835,
  "completed_at": 1780213900,
  "error": {
    "message": "视频生成失败，请检查提示词和参考素材",
    "code": ""
  }
}
```

### 9.5 状态兼容表

创建响应和查询响应的状态格式可能不同。客户端应按下面的方式兼容：

| 状态 | 含义 | 客户端行为 |
| --- | --- | --- |
| `PENDING`、`queued` | 排队中 | 等待后继续查询 |
| `RUNNING`、`in_progress` | 生成中 | 等待后继续查询 |
| `SUCCEEDED`、`completed` | 已完成 | 读取视频地址并停止查询 |
| `FAILED`、`FAILED: 原因`、`failed` | 已失败 | 记录错误并停止查询 |

判断状态时建议忽略大小写；若状态以 `FAILED:` 开头，冒号后的文本可作为失败原因。

## 10. JavaScript 完整轮询示例

以下示例应运行在服务端环境。请勿将 API Token 放入浏览器代码。

```javascript
const baseURL = 'https://ghlink.top';
const apiKey = process.env.GHLINK_API_KEY;

const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));

async function createVideo() {
  const response = await fetch(`${baseURL}/v1/video/async-generations`, {
    method: 'POST',
    headers: {
      Authorization: `Bearer ${apiKey}`,
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({
      model: '',
      prompt: '一只橘猫在草地上奔跑，电影级光影，镜头平稳跟拍',
      duration: 10,
      ratio: '16:9',
      resolution: '720p',
    }),
  });

  const body = await response.json();
  if (!response.ok || !body.task_id) {
    throw new Error(body?.error?.message || `创建任务失败：HTTP ${response.status}`);
  }
  return body.task_id;
}

async function waitForVideo(taskId) {
  const deadline = Date.now() + 10 * 60 * 1000;

  while (Date.now() < deadline) {
    await sleep(20_000);

    const response = await fetch(
      `${baseURL}/v1/video/async-generations/${encodeURIComponent(taskId)}`,
      { headers: { Authorization: `Bearer ${apiKey}` } },
    );
    const body = await response.json();

    if (!response.ok) {
      throw new Error(body?.error?.message || `查询任务失败：HTTP ${response.status}`);
    }

    const status = String(body.status || '').toLowerCase();
    if (status === 'completed' || status === 'succeeded') {
      const videoURL = body?.metadata?.url || body?.url;
      if (!videoURL) throw new Error('任务已完成，但响应中没有视频地址');
      return videoURL;
    }

    if (status === 'failed' || status.startsWith('failed:')) {
      throw new Error(body?.error?.message || body.status || '视频生成失败');
    }
  }

  throw new Error('等待视频生成超时');
}

const taskId = await createVideo();
const videoURL = await waitForVideo(taskId);
console.log(videoURL);
```

## 11. HTTP 错误

接口返回非 `2xx` HTTP 状态码时，请优先读取响应中的错误消息。

| HTTP 状态码 | 常见原因 | 建议处理方式 |
| --- | --- | --- |
| `400` | 参数缺失、时长或画幅不支持、素材数量超限、URL 格式错误 | 修改请求后重新提交 |
| `401` | Token 缺失、无效或已过期 | 检查 `Authorization` 请求头 |
| `403` | Token 没有模型或分组访问权限 | 联系管理员开通权限 |
| `404` | 路径错误或任务不存在 | 检查接口路径和 `task_id` |
| `429` | 请求过于频繁、并发受限或额度不足 | 降低频率并检查余额；按响应提示重试 |
| `500` | 平台内部异常 | 保存响应和任务 ID，稍后重试或联系技术支持 |
| `502`、`503`、`504` | 服务节点异常或上游超时 | 使用指数退避重试；避免立即高频重复提交 |

参数错误响应示例：

```json
{
  "error": {
    "message": "请求参数不合法，请检查 duration、ratio 和素材地址",
    "type": "invalid_request_error",
    "code": "invalid_request"
  }
}
```

> 不要对所有错误都自动重新创建任务。只有网络错误或明确可重试的 `429`、`502`、`503`、`504` 适合有限次数重试。对于已经返回 `task_id` 的请求，应继续查询原任务，避免重复任务和重复计费。

## 12. 常见问题

### 为什么任务创建后没有立即返回视频？

视频生成是异步操作。创建接口只负责返回 `task_id`，客户端需要通过查询接口获取最终结果。

### 可以每秒查询一次吗？

不可以。该模型的查询间隔应不低于 20 秒。高频查询可能触发限流，也不会让任务更快完成。

### 为什么素材 URL 在浏览器中能打开，接口却读取失败？

常见原因包括链接依赖 Cookie、需要登录、限制服务端访问、存在防盗链、重定向过多或签名链接已经过期。建议使用允许公网直接下载的对象存储或 CDN 地址。

### 为什么任务生成失败？

常见原因包括提示词或素材触发内容安全策略、素材不可访问、媒体格式异常、素材数量超限，以及输入视频与输出时长之和超过限制。请先简化提示词和素材后重新尝试。

### 视频地址可以长期使用吗？

不可以。生成结果仅临时保存约 10 小时。业务系统应在任务完成后立即把视频下载并保存到自己的存储空间。

## 13. 上线检查清单

- Base URL 使用 `https://ghlink.top`，不要重复拼接 `/v1`。
- `Authorization` 中的 `Bearer` 与 Token 之间保留一个空格。
- `model` 使用模型广场中的完整公开名称。
- 使用 `start_image_url` 和 `end_image_url` 传递首尾帧。
- 所有参考素材均为可直接访问的公网 HTTP/HTTPS 地址。
- 图片尺寸不低于 `300 × 300`。
- 图片、视频、音频数量以及总素材数量均未超限。
- 输入视频时长与输出视频时长之和不超过 25 秒。
- 两次任务状态查询之间至少间隔 20 秒。
- 客户端同时兼容大写和小写任务状态。
- 任务完成后立即下载并保存视频结果。
- 日志中记录公开 `task_id`，但不要记录完整 API Token。
