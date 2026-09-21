# GHLINK 下游 API 对接说明（给 Codex）

> 文档核对时间：2026-09-20
>
> 这是一份面向下游开发者和 Codex 的对接说明。只描述网站公开的调用方式，不包含上游供应商地址、渠道名称、内部模型映射、渠道密钥、后台配置或真实 API Key。

## 0. 给 Codex 的执行约束

实现下游客户端时，请严格遵守以下规则：

1. 将 `https://ghlink.top` 作为 API Base URL；如部署地址不同，将其替换为实际网站域名。
2. API Key 只能从环境变量或服务端密钥管理读取，不能写死在前端、公开仓库或示例代码中。
3. `model` 必须使用本文列出的完整字符串，大小写、数字、连字符和括号都不能自行修改。
4. 在真正调用前，优先请求 `GET /v1/models`；如果实时返回的模型列表与本文不同，以实时返回为准。
5. 当前模型广场显示的端点统计为：Chat 15、图片 5、视频 0。不要因为模型名称或卡片描述包含“视频”“音频”等字样，就自行改用视频接口。
6. 不要向请求中添加本文没有声明、且模型详情页没有声明的参数。未知能力应先返回明确错误或要求配置，而不是猜测字段名。

## 1. 认证与通用请求格式

所有请求均使用 Bearer Token：

```http
Authorization: Bearer <YOUR_API_KEY>
```

JSON 请求使用：

```http
Content-Type: application/json
```

推荐的环境变量：

```bash
export GHLINK_BASE_URL="https://ghlink.top"
export GHLINK_API_KEY="<YOUR_API_KEY>"
```

不要把真实 Token 写入本文件；下游拿到文档后，应由部署人员自行配置 `GHLINK_API_KEY`。

## 2. 当前公开模型清单

以下是 2026-09-20 在模型广场实际显示的 15 个模型。模型名称必须原样传递。

### 2.1 Chat 端点模型（15 个）

这些模型当前都按 Chat 端点公开，调用接口为 `POST /v1/chat/completions`：

```text
官方h3-1080p
官方h3-2k
d2.5(满血原生过脸30-10-10/480P)
gpt-image-2
gpt-image-2.5-flare
gpt-image-2.5-sunburst
mx-h3
nano-banana-pro
nano-banana2
sd2.0(满血9-3-3不卡脸720P)
Seedance2.0-c
Seedance2.0-fast-c
Seedance2.0-fast2-c
Seedance2.5-c
wan3.0(图片+音频参考)
```

### 2.2 图片生成端点模型（5 个）

以下 5 个模型在模型详情页同时公开了 `image-generation` 端点，图片生成调用 `POST /v1/images/generations`：

```text
gpt-image-2
gpt-image-2.5-flare
gpt-image-2.5-sunburst
nano-banana-pro
nano-banana2
```

图片模型也出现在 Chat 模型统计中。需要生成图片时，应使用图片端点；需要进行对话式调用时，才使用 Chat 端点。

## 3. 模型发现

### 3.1 请求

```bash
curl "$GHLINK_BASE_URL/v1/models" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

### 3.2 使用规则

- 启动时可以缓存模型列表，但不要永久写死。
- 如果模型不存在、被禁用或 Token 无权访问，应直接展示 API 返回的错误。
- 不要把供应商、渠道、倍率、上游模型名写入下游业务配置。
- 本文的模型清单是核对时点快照，实时 `/v1/models` 优先级更高。

## 4. Chat Completions

### 4.1 接口

```http
POST /v1/chat/completions
```

### 4.2 最小请求

```bash
curl "$GHLINK_BASE_URL/v1/chat/completions" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mx-h3",
    "messages": [
      {
        "role": "user",
        "content": "请用一句话介绍你自己。"
      }
    ]
  }'
```

详情页对 Chat 模型给出的调用结构是 OpenAI Chat Completions 结构；`messages` 是消息数组，`role` 常用 `system`、`user`、`assistant`。

### 4.3 通用参数

以下参数出现在当前 Chat 模型详情页的“支持的参数”表中。除必需字段外，建议只在业务确实需要时传递。

| 参数 | 类型 | 页面记录的默认值/范围 | 说明 |
| --- | --- | --- | --- |
| `model` | string | — | 本文模型 ID，必填 |
| `messages` | array | — | 对话消息，必填 |
| `temperature` | number | 默认 `1`，`0`～`2` | 采样温度 |
| `top_p` | number | 默认 `1`，`0`～`1` | 核采样累计概率 |
| `max_tokens` | integer | `>= 1` | 最大输出 Token 数 |
| `frequency_penalty` | number | 默认 `0`，`-2`～`2` | 惩罚高频 Token 重复 |
| `presence_penalty` | number | 默认 `0`，`-2`～`2` | 鼓励引入新话题 |
| `stop` | array | 最多 4 个字符串 | 停止生成字符串 |
| `seed` | integer | 页面未给默认值 | 尽量保证可复现的采样种子 |
| `n` | integer | 默认 `1`，`>= 1` | 候选结果数量 |
| `stream` | boolean | 默认 `false` | 是否通过 SSE 流式返回 |
| `response_format` | object | 页面未给默认值 | JSON 对象或 Schema 输出约束 |
| `tools` | array | 页面未给默认值 | 工具/函数声明 |
| `tool_choice` | string | `auto`、`none`、`required` | 工具选择策略 |
| `logprobs` | boolean | 默认 `false` | 是否返回 Token 对数概率 |
| `top_logprobs` | integer | `0`～`20` | 每个 Token 返回的 Top 概率数量 |
| `logit_bias` | object | 页面未给默认值 | Token 偏置映射 |
| `user` | string | 页面未给默认值 | 用于风险审计的终端用户标识 |

模型卡片中的能力描述不等于参数保证。若某模型对参数返回 `400`，客户端应展示错误并允许移除该可选参数重试；不要默默改写模型名或接口。

### 4.4 非流式响应

```json
{
  "id": "chatcmpl_xxx",
  "object": "chat.completion",
  "created": 1780213835,
  "model": "mx-h3",
  "choices": [
    {
      "index": 0,
      "message": {
        "role": "assistant",
        "content": "你好！"
      },
      "finish_reason": "stop"
    }
  ],
  "usage": {
    "prompt_tokens": 12,
    "completion_tokens": 4,
    "total_tokens": 16
  }
}
```

客户端应从 `choices[0].message.content` 读取文本，并兼容 `usage` 缺失的情况。

### 4.5 流式响应

请求中传入 `"stream": true` 后，响应为 SSE：

```bash
curl "$GHLINK_BASE_URL/v1/chat/completions" \
  -N \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mx-h3",
    "messages": [{"role": "user", "content": "写一句欢迎语。"}],
    "stream": true
  }'
```

客户端必须按行解析 `data:` 事件，收到 `data: [DONE]` 后结束读取。不要等待整个 HTTP body 下载完毕后才显示内容。

## 5. 图片生成

### 5.1 接口

```http
POST /v1/images/generations
```

### 5.2 最小请求

```bash
curl "$GHLINK_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一张极简风格的日落海报",
    "size": "1024x1024",
    "n": 1,
    "response_format": "url"
  }'
```

### 5.3 当前详情页确认的参数

| 参数 | 类型 | 页面记录的默认值/范围 | 说明 |
| --- | --- | --- | --- |
| `model` | string | — | 图片模型 ID，必填 |
| `prompt` | string | 必填 | 图片文字描述 |
| `size` | enum | 默认 `1024x1024` | 输出尺寸 |
| `quality` | enum | 默认 `standard` | 生成质量预设 |
| `style` | enum | 默认 `vivid` | 画风 |
| `n` | integer | 默认 `1`，`1`～`10` | 生成图片数量 |
| `response_format` | enum | 默认 `url` | 图片结果返回方式 |

当前详情页只明确记录了上述字段和默认值。虽然模型卡片描述中提到部分模型支持参考图或多种尺寸，但没有在该参数表中给出统一的字段名和取值范围；因此下游 Codex 实现不得自行发明 `reference_images`、`image_urls` 等字段，也不要在未验证前承诺图片编辑或多图输入能力。

### 5.4 图片响应

```json
{
  "created": 1780213835,
  "data": [
    {
      "url": "https://example.invalid/generated-image.png"
    }
  ]
}
```

客户端读取 `data[].url`。示例中的域名仅为占位符，不要将其写入生产配置。

## 6. Python SDK 示例

```python
import os
from openai import OpenAI

client = OpenAI(
    api_key=os.environ["GHLINK_API_KEY"],
    base_url=f'{os.getenv("GHLINK_BASE_URL", "https://ghlink.top")}/v1',
)

response = client.chat.completions.create(
    model="mx-h3",
    messages=[{"role": "user", "content": "你好"}],
)
print(response.choices[0].message.content)
```

图片调用：

```python
result = client.images.generate(
    model="gpt-image-2",
    prompt="一张极简风格的日落海报",
    size="1024x1024",
    n=1,
    response_format="url",
)
print(result.data[0].url)
```

## 7. JavaScript / TypeScript SDK 示例

```ts
import OpenAI from "openai";

const client = new OpenAI({
  apiKey: process.env.GHLINK_API_KEY,
  baseURL: `${process.env.GHLINK_BASE_URL ?? "https://ghlink.top"}/v1`,
});

const completion = await client.chat.completions.create({
  model: "Seedance2.0-c",
  messages: [{ role: "user", content: "你好" }],
});

console.log(completion.choices[0]?.message?.content ?? "");
```

流式调用：

```ts
const stream = await client.chat.completions.create({
  model: "mx-h3",
  messages: [{ role: "user", content: "写一句欢迎语。" }],
  stream: true,
});

for await (const chunk of stream) {
  process.stdout.write(chunk.choices[0]?.delta?.content ?? "");
}
```

## 8. 错误处理与重试

错误通常为 OpenAI 风格结构：

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

建议按下面方式处理：

| HTTP 状态 | 客户端行为 |
| --- | --- |
| `400` | 修正模型名或请求参数后再提交，不要盲目重试 |
| `401` | 检查 Token 是否缺失或无效 |
| `403` | 检查 Token 是否有该模型权限 |
| `404` | 检查路径和模型 ID；不要自动切换到上游地址 |
| `429` | 降低并发，遵循 `Retry-After`（如响应提供），使用指数退避 |
| `5xx` | 有限次数指数退避；记录请求 ID 和错误信息 |

如果一次请求已经返回业务结果，不要因为客户端超时就立即重复提交；优先使用原请求的结果或重新查询业务侧保存的请求记录，避免重复执行。

## 9. 不应写入下游代码的内容

- 上游供应商名称、上游 URL、渠道编号、渠道密钥。
- 内部模型映射名、内部路由分组、管理员价格或倍率。
- 真实 API Key、Cookie、登录态、浏览器存储内容。
- 未在模型详情页或实时模型接口中声明的参数和端点。
- 将当前卡片描述直接当作视频生成协议；当前公开端点统计中视频为 0。

## 10. Codex 对接验收清单

- [ ] Base URL 和 API Key 通过环境变量配置。
- [ ] 启动时调用 `GET /v1/models` 并能处理模型变更。
- [ ] 15 个模型 ID 按原样保留。
- [ ] Chat 请求使用 `/v1/chat/completions`。
- [ ] 图片生成请求使用 `/v1/images/generations`。
- [ ] 支持非流式响应和 `stream: true` 的 SSE 响应。
- [ ] `data: [DONE]` 能正确结束流式读取。
- [ ] 错误信息、HTTP 状态和请求 ID（如有）会保留到日志，但不会记录 Token。
- [ ] 未实现的视频、图片编辑、参考图扩展能力不会被伪造为已支持。
- [ ] 集成测试使用占位 Token，并由部署人员在运行环境注入真实 Token。

## 11. 媒体转存第三阶段（管理员运维接口）

视频成功后，网站先把上游媒体下载到持久化转存队列，再按图床优先级上传并校验公开 URL。转存任务使用原任务 ID 做唯一约束；转存失败不会再次提交上游生成请求，也不会自动退款已经发生的上游成本。

以下接口仅供 Root 管理员使用，不能放入普通下游客户端：

```http
GET /api/option/media_storage/transfers?page=1&page_size=20&status=FAILED
POST /api/option/media_storage/transfers/{job_id}/retry
```

监控接口返回队列深度、各状态数量、近期失败率、平均转存耗时、图床熔断状态和脱敏任务列表；不会返回加密保存的上游 URL、上游认证信息、租约令牌或原始上游响应。手动重试只把 `FAILED`/`RETRY` 任务重新排队，复用已经下载的暂存文件和原始任务，不重新扣费、不重新生成。

图床连续 3 次上传或公开 URL 校验失败会在当前 Worker 进程内熔断 2 分钟，期间自动跳过该图床并尝试其他已启用图床；下载上游媒体失败不会错误熔断图床。生产环境应至少配置两个不同域名/服务商的图床，并为 Worker 配置共享的 `MEDIA_SPOOL_DIR` 持久卷，否则 Worker 重启或跨实例接管时无法复用暂存文件。
