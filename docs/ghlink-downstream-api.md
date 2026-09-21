# GHLINK 下游 API 对接文档

> 文档版本：2026-09-21
>
> 本文只描述 GHLINK 对下游公开的接口、模型 ID 和调用流程。不要把渠道名称、内部模型映射、内部任务号、供应商地址或密钥写入下游业务配置。

## 1. 接入信息

### 1.1 Base URL

```text
https://ghlink.top
```

如果部署域名发生变化，只替换 Base URL；接口路径保持不变。

### 1.2 认证

所有请求都使用 Bearer Token：

```http
Authorization: Bearer <YOUR_API_KEY>
```

JSON 请求还需要：

```http
Content-Type: application/json
```

API Key 只能放在服务端环境变量或密钥管理系统中，不要写入浏览器代码、移动端安装包或公开仓库。

建议配置：

```bash
export GHLINK_BASE_URL="https://ghlink.top"
export GHLINK_API_KEY="<YOUR_API_KEY>"
```

## 2. 模型发现与模型 ID 规则

### 2.1 启动时查询模型

模型可能因为权限、分组、库存或线路调整而变化。每次启动或缓存过期时，先查询当前 Key 可用的模型：

```bash
curl "$GHLINK_BASE_URL/v1/models" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

成功响应使用 `data[].id` 提供可用模型，例如：

```json
{
  "success": true,
  "object": "list",
  "data": [
    {
      "id": "Seedance2.0-c",
      "object": "model"
    }
  ]
}
```

只允许调用接口实时返回的 `id`。本文中的模型清单是当前线上映射的快照，不替代实时返回结果。

### 2.2 模型 ID 必须原样传递

- 大小写、连字符、数字、中文、括号都属于模型 ID。
- 不要把模型 ID 改成内部映射名。
- 不要根据模型名称自行拼接供应商前缀或渠道前缀。
- 如果同一个模型名在不同文档中出现不同端点，以当前 `/v1/models` 和模型详情页显示的公开端点为准。

## 3. 当前线上模型与公开端点

以下是根据当前模型广场截图和已提供接口文档整理的公开模型快照。

### 3.1 视频模型

这些模型统一使用异步视频任务（视频创建与查询）接口：

| 公开模型 ID | 创建接口 | 查询接口 |
| --- | --- | --- |
| `官方h3-1080p` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `官方h3-2k` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `d2.5(满血原生过脸30-10-10/480P)` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `mx-h3` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `sd2.0(满血9-3-3不卡脸720P)` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `Seedance2.0-c` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `Seedance2.0-fast-c` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `Seedance2.0-fast2-c` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `Seedance2.5-c` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |
| `wan3.0(图片+音频参考)` | `POST /v1/videos` | `GET /v1/videos/{task_id}` |

视频任务是异步的：创建请求只代表任务已受理，必须保存 `task_id` 并轮询查询。

### 3.2 图片模型

当前线上图片模型使用同步图片生成接口，不使用异步图片接口：

| 公开模型 ID | 创建接口 | 是否需要轮询 |
| --- | --- | --- |
| `gpt-image-2` | `POST /v1/images/generations` | 否 |
| `gpt-image-2.5-flare` | `POST /v1/images/generations` | 否 |
| `gpt-image-2.5-sunburst` | `POST /v1/images/generations` | 否 |
| `nano-banana-pro` | `POST /v1/images/generations` | 否 |
| `nano-banana2` | `POST /v1/images/generations` | 否 |

注意：其他渠道文档中也出现过 `nano-banana*` 和 `gpt-image2` 的异步图片接口。那是独立的渠道预设，不能覆盖当前线上模型的同步图片映射；实际调用时以当前 Key 的 `/v1/models` 和模型详情页为准。

## 4. 视频生成接口

### 4.1 创建任务

```http
POST /v1/videos
```

完整请求示例：

```bash
curl -X POST "$GHLINK_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seedance2.0-c",
    "prompt": "一只橘猫在阳光下的草地上慢慢行走，电影感镜头，画面稳定",
    "duration": 5,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

如果当前模型的公开配置声明支持幂等，可额外添加：

```http
Idempotency-Key: order-20260921-0001
```

如果当前模型的公开配置支持 `Idempotency-Key`，建议使用业务订单的唯一值。网络超时后重试创建请求时，复用同一个值，不要换一个新值重复创建。即使未声明支持幂等，也应先保存业务订单和已返回的 `task_id`；创建请求超时后先查询任务，不要直接盲目再次创建。

### 4.2 视频请求字段

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `/v1/models` 返回的公开视频模型 ID |
| `prompt` | string | 是 | 视频内容、动作和镜头描述 |
| `duration` | integer | 按模型 | 时长，具体范围以模型详情为准 |
| `ratio` | string | 否 | 例如 `16:9`、`9:16`、`1:1` |
| `resolution` | string | 否 | 例如 `480p`、`720p`、`1080p`，以模型支持范围为准 |
| `referenceImages` | string[] | 否 | 参考图片公网 URL |
| `referenceVideos` | string[] | 否 | 参考视频公网 URL |
| `referenceAudios` | string[] | 否 | 参考音频公网 URL |
| `first_frame_image` | string | 否 | 部分模型支持的首帧图片 URL |
| `last_frame_image` | string | 否 | 部分模型支持的尾帧图片 URL |

建议只发送当前模型支持的字段。不要为了兼容其他渠道，把 `extra`、内部模型名、内部任务号或不属于当前模型的参数混入请求。

### 4.3 参考素材规则

- URL 必须是服务器可以直接访问的公网 `http(s)` 地址。
- 不要使用依赖 Cookie、登录态、内网地址或短时间即过期的 URL。
- 参考图数组中的第 N 项，如果在提示词中被引用，应使用对应的 `@图N` 标记。
- 首尾帧字段和参考图数组是两套机制；使用首尾帧时不要在提示词中虚构不存在的 `@图N`。
- 每个模型支持的参考图、参考视频、参考音频数量可能不同，以模型详情页为准。

### 4.4 创建响应

创建成功后保存 `task_id`。客户端应同时兼容 `id` 和 `task_id`，优先使用 `task_id`：

```json
{
  "id": "task_xxxxxxxxx",
  "task_id": "task_xxxxxxxxx",
  "object": "video",
  "status": "queued",
  "progress": 0
}
```

如果只返回 `id`，将 `id` 作为查询任务 ID。

## 5. 视频轮询、结果和下载

### 5.1 查询任务

```http
GET /v1/videos/{task_id}
```

```bash
curl "$GHLINK_BASE_URL/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $GHLINK_API_KEY"
```

建议每 5–10 秒查询一次。如果创建响应或模型详情给出更长的 `next_poll_seconds`，应使用更长的间隔。

### 5.2 状态处理

| 状态 | 客户端处理 |
| --- | --- |
| `queued` | 继续查询，不要重复创建 |
| `processing` / `in_progress` / `running` | 继续查询 |
| `completed` / `succeeded` | 停止轮询，读取结果 URL |
| `failed` / `error` / `cancelled` | 停止轮询，记录错误；修正参数后创建新任务 |

查询请求临时返回 `408`、`409`、`425`、`429` 或 `5xx`，不等于任务失败。保留同一个 `task_id`，按退避策略继续查询。

### 5.3 读取结果 URL

不同模型的完成响应字段可能略有差异，客户端按以下顺序读取第一个非空值：

1. `video_url`
2. `result_url`
3. `url`
4. `metadata.url`
5. `video.url`

示例：

```json
{
  "id": "task_xxxxxxxxx",
  "task_id": "task_xxxxxxxxx",
  "status": "completed",
  "progress": 100,
  "video_url": "https://ghlink.top/v1/videos/task_xxxxxxxxx/content"
}
```

只使用查询响应返回的结果地址。不要自行拼接上游地址、内部地址或任务下载地址；如果任务完成但没有结果 URL，应继续查询或联系平台。

### 5.4 Python 轮询示例

```python
import os
import time
import requests

BASE_URL = os.environ.get("GHLINK_BASE_URL", "https://ghlink.top")
API_KEY = os.environ["GHLINK_API_KEY"]


def create_video(model: str, prompt: str) -> dict:
    response = requests.post(
        f"{BASE_URL}/v1/videos",
        headers={
            "Authorization": f"Bearer {API_KEY}",
            "Content-Type": "application/json",
        },
        json={
            "model": model,
            "prompt": prompt,
            "duration": 5,
            "ratio": "16:9",
            "resolution": "720p",
        },
        timeout=60,
    )
    response.raise_for_status()
    return response.json()


def poll_video(task_id: str, timeout_seconds: int = 1800) -> dict:
    deadline = time.time() + timeout_seconds
    while time.time() < deadline:
        response = requests.get(
            f"{BASE_URL}/v1/videos/{task_id}",
            headers={"Authorization": f"Bearer {API_KEY}"},
            timeout=30,
        )

        if response.status_code in {408, 409, 425, 429} or 500 <= response.status_code < 600:
            time.sleep(10)
            continue

        response.raise_for_status()
        data = response.json()
        status = str(data.get("status", "")).lower()

        if status in {"completed", "succeeded"}:
            for key in ("video_url", "result_url", "url"):
                if data.get(key):
                    return {"task": data, "url": data[key]}
            metadata = data.get("metadata") or {}
            if metadata.get("url"):
                return {"task": data, "url": metadata["url"]}
            video = data.get("video") or {}
            if video.get("url"):
                return {"task": data, "url": video["url"]}
            raise RuntimeError("task completed but no result URL was returned")

        if status in {"failed", "error", "cancelled"} or status.startswith("failed:"):
            raise RuntimeError(data.get("error") or data)

        time.sleep(10)

    raise TimeoutError(f"video task timed out: {task_id}")


created = create_video(
    "Seedance2.0-c",
    "一只橘猫在阳光下的草地上慢慢行走，电影感镜头",
)
task_id = created.get("task_id") or created.get("id")
if not task_id:
    raise RuntimeError(f"creation response did not contain task_id: {created}")

result = poll_video(task_id)
print(result["url"])
```

## 6. 图片生成接口

### 6.1 创建图片

```http
POST /v1/images/generations
```

```bash
curl -X POST "$GHLINK_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $GHLINK_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一张简洁的海边日落电影海报",
    "aspect_ratio": "1:1",
    "resolution": "1K",
    "n": 1
  }'
```

字段：

| 字段 | 类型 | 必填 | 说明 |
| --- | --- | --- | --- |
| `model` | string | 是 | `/v1/models` 返回的公开图片模型 ID |
| `prompt` | string | 是 | 图片描述 |
| `aspect_ratio` | string | 否 | 例如 `1:1`、`16:9`、`9:16` |
| `resolution` | string | 否 | 例如 `1K`、`2K`、`4K`，以模型为准 |
| `n` | integer | 否 | 生成数量，以模型详情为准 |

图片生成响应可能返回 URL 或 Base64：

```json
{
  "data": [
    {
      "url": "https://example.com/generated-image.png"
    }
  ]
}
```

客户端应同时兼容 `data[].url` 和 `data[].b64_json`。

## 7. 文档预设模型族

下面的端点来自你提供的其他渠道文档，作为未接入模型的预设参考。它们不是当前 15 个线上模型的替代映射；只有当模型 ID 实际出现在当前 Key 的 `/v1/models` 返回中时，才可以调用。

| 预设模型族或完整 ID | 创建接口 | 查询接口 | 备注 |
| --- | --- | --- | --- |
| 明确标记为官方转接线路的模型 | `POST /v1/video/generations` | `GET /v1/video/generations/{task_id}` | 只有模型详情明确显示该端点时使用 |
| `004系列/*` | `POST /v1/videos` | `GET /v1/videos/{task_id}` | 模型 ID 必须包含完整前缀 |
| `omni-video-1`、`omni-video-1-fast`、`omni-video-1-pro` | `POST /v1/videos` | `GET /v1/videos/{task_id}` | 取消能力不作为统一公共接口；只有实时模型详情明确提供时才使用 |
| `sora2`、`veo31*`、`kling-v3`、`grok-imagine-video` | `POST /v1/video/async-generations` | `GET /v1/video/async-generations/{task_id}` | 异步视频预设，当前截图未列出 |
| `nano-banana*`、`gpt-image2`（异步渠道版本） | `POST /v1/images/async-generations` | `GET /v1/images/async-generations/{task_id}` | 仅在实时模型目录和详情页都声明时使用 |

同名模型可能在不同渠道文档中拥有不同端点。因此集成程序必须按以下优先级决策：

1. 当前 Key 的 `GET /v1/models` 和模型详情页明确声明的端点。
2. 当前模型的显式公开配置。
3. 本节预设。

不能仅凭 `nano-banana2`、`wan3.0` 或其他模型名称猜测异步接口。

## 8. 兼容路径

对于统一视频任务，以下路径在兼容配置中可能可用，但新接入建议使用规范路径：

| 规范用途 | 建议路径 | 兼容路径 |
| --- | --- | --- |
| 创建视频 | `POST /v1/videos` | `POST /v1/videos/generations`、`POST /v1/video/generations` |
| 查询视频 | `GET /v1/videos/{task_id}` | `GET /v1/video/generations/{task_id}`、`GET /v1/videos/generations/{task_id}` |

不要同时轮询多个等价路径。选择一个路径并持续使用同一个 `task_id`。

## 9. 错误处理

| 状态码 | 常见含义 | 建议 |
| --- | --- | --- |
| `400` | 参数、模型或请求体错误 | 先核对模型 ID 和字段名 |
| `401` | API Key 无效或已撤销 | 更换或重新生成 Key |
| `403` | 当前 Key 或分组无权使用模型 | 重新查询 `/v1/models` |
| `402` | 余额或积分不足 | 检查账户余额 |
| `404` | 路径、任务或资源不存在 | 核对规范路径和 `task_id` |
| `408` / `409` / `425` | 查询暂时不可用 | 保留任务 ID，稍后继续查询 |
| `429` | 频率限制 | 按 `Retry-After` 或退避时间重试 |
| `500` / `502` / `503` | 服务或可用线路暂时异常 | 不要重复创建，视频任务继续查询；新任务按退避重试 |

错误排查时只提交以下信息：公开模型 ID、接口路径、时间、`task_id` 和脱敏后的错误内容。不要提交完整 API Key。

## 10. 给 Codex 的接入要求

实现下游客户端时，请让 Codex 遵守以下规则：

1. 从环境变量读取 `GHLINK_BASE_URL` 和 `GHLINK_API_KEY`。
2. 启动时先调用 `GET /v1/models`，只使用实时返回的公开模型 ID。
3. 当前 15 个模型按本文第 3 节选择视频或同步图片接口。
4. 视频创建后只保存一个 `task_id`，不要重复创建任务。
5. 轮询必须使用创建时相同的 API Key。
6. 完成后从响应中的 URL 字段取结果，不要猜测或拼接内部地址。
7. 预设模型只能在实时目录和模型详情页确认后启用。
8. 不要在代码、日志、前端或文档中写入内部映射模型名、渠道地址或密钥。
