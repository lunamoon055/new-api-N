# GHLINK 下游对接快速修复版

更新日期：2026-09-20

## 先修复当前 404

截图中的错误请求是：

```text
POST /v1/images/async-generations
```

当前网站没有这个图片接口，所以会返回 `HTTP 404: Invalid URL`。

请把下游配置中的图片地址改为：

```text
POST /v1/images/generations
```

不要把图片接口写成 `/v1/images/async-generations`。图床转存不会改变图片接口路径；图床只负责在生成成功后替换 `data[].url` 的地址。

## 推荐配置

```env
GHLINK_BASE_URL=https://ghlink.top
GHLINK_API_KEY=sk-替换为你的令牌
GHLINK_IMAGE_ENDPOINT=/v1/images/generations
GHLINK_VIDEO_SUBMIT_ENDPOINT=/v1/video/async-generations
GHLINK_VIDEO_QUERY_ENDPOINT=/v1/video/async-generations/{task_id}
```

请求头统一使用：

```http
Authorization: Bearer sk-替换为你的令牌
Content-Type: application/json
```

不要把图床 Token 放到下游请求中。下游只使用 GHLINK API Token。

## 图片生成：同步接口

### 最小请求

```bash
curl -X POST "https://ghlink.top/v1/images/generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image2",
    "prompt": "一只坐在窗边的橘猫",
    "size": "1024x1024",
    "n": 1,
    "response_format": "url"
  }'
```

### 成功响应

```json
{
  "created": 1780213835,
  "data": [
    {
      "url": "https://media.ghlink.top/file/xxx",
      "b64_json": "",
      "revised_prompt": ""
    }
  ]
}
```

下游读取 `data[0].url`。如果模型只返回 Base64，则读取 `data[0].b64_json`。配置图床后，字段结构不变，只是 `url` 可能变成图床地址；图床失败时仍会保留上游 URL。

## 视频生成：异步接口

图片和视频不要共用同一个 endpoint。

### 1. 提交任务

```bash
curl -X POST "https://ghlink.top/v1/video/async-generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "video-2.5",
    "prompt": "清晨海边公路，电影级航拍",
    "duration": 8,
    "aspect_ratio": "16:9",
    "resolution": "720p"
  }'
```

保存响应中的 `task_id`：

```json
{
  "id": "task_xxx",
  "task_id": "task_xxx",
  "status": "queued",
  "progress": 0
}
```

### 2. 查询任务

```bash
curl "https://ghlink.top/v1/video/async-generations/task_xxx" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

轮询到 `status` 为 `completed` 或 `succeeded` 后，按以下顺序读取结果：

```text
metadata.url -> url
```

失败时读取：

```text
error.message
```

建议轮询间隔为 10–20 秒，不要高频请求。

## 下游代码最小修改

如果下游代码当前是：

```js
const endpoint = `${baseUrl}/v1/images/async-generations`
```

直接改为：

```js
const endpoint = `${baseUrl}/v1/images/generations`
```

如果下游同时调用视频，保持视频地址为：

```js
const submitEndpoint = `${baseUrl}/v1/video/async-generations`
const queryEndpoint = (taskId) =>
  `${baseUrl}/v1/video/async-generations/${encodeURIComponent(taskId)}`
```

## 错误快速定位

| 状态 | 常见原因 | 修复 |
| --- | --- | --- |
| `404 Invalid URL` | 路径写成 `/v1/images/async-generations` | 改成 `/v1/images/generations` |
| `401` | GHLINK API Token 缺失或无效 | 检查 `Authorization: Bearer ...` |
| `400` | 缺少 `model` 或 `prompt`，或请求体字段不支持 | 先使用本文最小请求验证 |
| `429` | 请求频率过高 | 降低并发和轮询频率 |
| 图片已成功但 URL 仍是上游地址 | 图床上传失败 | 检查后台图床 Token、`Authorization`、`0.src` 配置 |

## 不要使用的路径

```text
/v1/images/async-generations       # 不存在
/v1/images/variations               # 当前未实现
/v1/creation/images/async-generations # 当前不作为图片异步接口
```

## 验收顺序

1. 先用本文图片 curl 请求确认 `POST /v1/images/generations` 返回 200。
2. 确认响应存在 `data[0].url` 或 `data[0].b64_json`。
3. 再测试视频提交和查询接口。
4. 最后在网站后台单独测试图床；图床失败不应改变下游图片接口路径和响应结构。

