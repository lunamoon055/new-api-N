# GHLINK 模型对接文档（基于代码分析）

**重要说明**：本文档基于 new-api 项目源代码分析生成，包含真实的 API 接口路径和模型名称。

---

## 🔌 API 接口路径（真实）

根据 `router/video-router.go` 和 `router/api-router.go`，项目支持以下接口：

### 视频生成接口

#### 1. 异步视频生成（推荐）
```
POST /v1/video/async-generations       # 提交任务
GET  /v1/video/async-generations/:task_id  # 查询任务状态
```

#### 2. 同步视频生成
```
POST /v1/video/generations
GET  /v1/video/generations/:task_id
```

#### 3. OpenAI 兼容接口
```
POST /v1/videos/generations
GET  /v1/videos/generations/:task_id
POST /v1/videos
GET  /v1/videos/:task_id
```

#### 4. 视频 Remix
```
POST /v1/videos/:video_id/remix
```

### 图片生成接口

#### 标准接口（通过 relay-router）
```
POST /v1/images/generations      # 图片生成
```

#### Creation API（Dashboard使用）
```
POST /api/creation/images/generations
GET  /api/creation/images/generations/:task_id
POST /api/creation/video/async-generations
GET  /api/creation/video/async-generations/:task_id
```

---

## 📦 支持的模型列表（来自源代码）

根据 `relay/channel/task/sora/constants.go`，系统支持以下模型：

### 视频生成模型

#### Sora 系列
- `sora2`
- `sora-2`
- `sora-2-pro`

#### MiniMax H3 系列
- `minimax-h3`
- `minimax-h3-480p`
- `minimax-h3-768p`
- `minimax-h3-2k`
- `minimax-h3-4k`

#### Video 2.0 系列
- `video-2.0`
- `video-2.0-fast`
- `video-2.0-mini`
- `video-2.0-480p`
- `video-2.0-fast-480p`
- `video-2.0-mini-480p`
- `video-2.5`
- `video-2.5-480p`

#### SD2 系列
- `sd2-mini`
- `sd2-fast`
- `sd2满血`
- `sd-2.0-933`
- `sd-2-c8`
- `sd2.0`
- `sd-mini`
- `sd-2.5`

#### Seedance 系列
- `seedance-2.0`
- `seedance-2.5`

#### Wan3.0 系列
- `wan3.0-480p`
- `wan3.0-720p`
- `wan3.0-1080p`
- `wan3.0-video`
- `wan3.0-video-prime`
- `wan3.0-image`
- `wan3.0-image-prime`

#### 其他视频模型
- `ko3`
- `kling-v3`
- `veo31`
- `veo31-fast`
- `veo31-ref`
- `grok-imagine-video`
- `grok-imagine-video-1.5`
- `omni-video-1`
- `omni-video-1-fast`
- `omni-video-1-pro`

#### 004系列（特殊配置模型）
- `004系列/minimax_h3(8图3音频)`
- `004系列/minimax-h3 2k`
- `004系列/minimax-h3 768p`
- `004系列/Minimax-H3(933/10-15秒768P)`
- `004系列/seedance-2.0-fast-deal(903不卡人脸480P)`
- `004系列/seedance-2.0-fast-deal(903不卡人脸720P)`
- `004系列/sd2.0(满血5-15秒720P)`
- `004系列/sd2.0(支持过人脸5/10秒720P)`
- `004系列/sd2.0(支持过人脸全系4-15秒720P)`
- `004系列/sd2.0(933/4-15秒支持过人脸720P)`
- `004系列/sd2.0(933/10-15秒720P)`
- `004系列/sd2.0(933/480p最多15秒/720p最多12秒)`
- `004系列/sd2.5(全系支持过人脸4-30秒/720P)`
- `004系列/sd2.5(全系支持过人脸4-30秒720P)`
- `004系列/sd2.5(原生过人脸9图720P)`
- `004系列/sd2.5(30图4-30秒480P/720P)`
- `004系列/sd2.5(10-10-10支持过人脸4-30秒720P)`
- `004系列/sd2.5(30-10-10支持过人脸4-30秒720P)`
- `004系列/video-editing(换脸/视频编辑必须上传1视频1图换脸用)`
- `004系列/grok(支持文生首帧图片参考)`
- `004系列/grok(文生首尾帧图2-7图参考)`

---

## 🔧 截图中模型的实际对应关系

根据您的截图和代码分析，以下是模型的真实名称：

| 截图显示 | 实际模型名称（代码中） | 类型 |
|---------|---------------------|------|
| 官方h3-1080p | `minimax-h3` 或 `004系列/...` | 视频 |
| 官方h3-2k | `minimax-h3-2k` 或 `004系列/minimax-h3 2k` | 视频 |
| d2.5(...) | `sd-2.5` 或 `004系列/sd2.5(...)` | 视频 |
| sd2.0(...) | `sd2.0` 或 `004系列/sd2.0(...)` | 视频 |
| Seedance2.0-c | `seedance-2.0` | 视频 |
| Seedance2.0-fast-c | `seedance-2.0` (fast变体) | 视频 |
| Seedance2.0-fast2-c | `seedance-2.0` (fast2变体) | 视频 |
| Seedance2.0-3-c | `seedance-2.5` | 视频 |
| wma3.0(...) | `wan3.0-480p` / `wan3.0-720p` / `wan3.0-1080p` | 视频 |
| gpt-image-2 | OpenAI 图片模型 | 图片 |
| gpt-image-2.5-flare | OpenAI 图片模型变体 | 图片 |
| gpt-image-2.5-sunburst | OpenAI 图片模型变体 | 图片 |
| mx-h3 | `minimax-h3` | 混合 |
| nano-banana-pro | 第三方图片模型 | 图片 |
| nano-banana2 | 第三方图片模型 | 图片 |

**⚠️ 重要提示**：
1. 截图中显示的是**前端展示名称**，可能与后端实际模型名称不同
2. 实际使用时，需要在渠道配置中查看**上游模型映射**
3. 模型名称必须与代码中 `ModelList` 中的名称完全一致

---

## 📝 正确的 API 请求示例

### 示例 1: MiniMax H3 视频生成

```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "minimax-h3-2k",
    "prompt": "一只可爱的猫咪在阳光下玩耍",
    "duration": 8,
    "aspect_ratio": "16:9"
  }'
```

**响应示例**：
```json
{
  "id": "task_abc123",
  "task_id": "task_abc123",
  "object": "video.generation",
  "model": "minimax-h3-2k",
  "status": "queued",
  "created_at": 1726905600
}
```

### 示例 2: 查询任务状态

```bash
curl https://ghlink.top/v1/video/async-generations/task_abc123 \
  -H "Authorization: Bearer sk-你的密钥"
```

**响应示例（处理中）**：
```json
{
  "id": "task_abc123",
  "status": "in_progress",
  "progress": 45
}
```

**响应示例（完成）**：
```json
{
  "id": "task_abc123",
  "status": "completed",
  "progress": 100,
  "url": "https://example.com/video.mp4",
  "video_url": "https://example.com/video.mp4",
  "completed_at": 1726905720
}
```

### 示例 3: SD2 系列视频生成

```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2.0",
    "prompt": "海边日落，电影级画质",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

### 示例 4: Wan3.0 多模态参考

```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "wan3.0-1080p",
    "prompt": "根据参考图片和音频生成视频",
    "duration": 10,
    "aspect_ratio": "16:9",
    "image_urls": ["https://example.com/image.jpg"],
    "audio_url": "https://example.com/audio.mp3"
  }'
```

---

## 🗂️ 模型名称查找步骤

### 步骤 1: 登录 GHLINK 管理后台
访问：`https://ghlink.top/dashboard`

### 步骤 2: 查看渠道配置
导航到：**控制台 -> 渠道管理**

### 步骤 3: 查看模型映射
在渠道详情中查看：
- **前端模型名称**：用户在界面看到的名称
- **后端模型名称**：实际发送给上游的模型名称

### 步骤 4: 确认使用的接口
根据模型类型选择正确的接口：
- **async-generations 模型**：`minimax-h3`, `video-2.0`, `wan3.0`, `sora2`, `kling-v3`, `veo31`, `grok-imagine-video`
- **其他模型**：可能使用 `/v1/video/generations` 或 `/v1/videos`

---

## ⚙️ 模型参数判断逻辑（来自源代码）

### 判断是否使用 async-generations 接口

根据 `adaptor.go` 中的代码：

```go
func isAsyncGenerationsModel(modelName string) bool {
    normalizedModelName := strings.ToLower(strings.TrimSpace(modelName))
    if isVideo2Model(normalizedModelName) || 
       isMiniMaxH3Model(normalizedModelName) || 
       isWan30Model(normalizedModelName) {
        return true
    }
    switch normalizedModelName {
    case "sora2", "sora-2", "kling-v3", "ko3", 
         "veo31", "veo31-fast", "veo31-ref", 
         "grok-imagine-video":
        return true
    default:
        return false
    }
}
```

### 判断是否为 Video 2.0 模型

```go
func isVideo2Model(modelName string) bool {
    return strings.HasPrefix(modelName, "video-2.")
}
```

### 判断是否为 MiniMax H3 模型

```go
func isMiniMaxH3Model(modelName string) bool {
    return modelName == "minimax-h3" || 
           strings.HasPrefix(modelName, "minimax-h3-")
}
```

---

## 🚨 常见错误和解决方案

### 错误 1: 模型名称不匹配
```json
{
  "error": {
    "message": "model not found",
    "code": "invalid_request"
  }
}
```

**原因**：使用了前端显示名称而非实际模型名称

**解决**：
1. 检查渠道配置中的模型映射
2. 使用代码 `ModelList` 中的确切名称
3. 注意大小写敏感

### 错误 2: 接口路径错误
```json
{
  "error": {
    "message": "not found",
    "code": "not_found"
  }
}
```

**原因**：使用了错误的接口路径

**解决**：
- MiniMax H3、Video 2.0、Wan3.0 → `/v1/video/async-generations`
- SD2、Seedance → 根据渠道配置，可能是 `/v1/videos` 或其他

### 错误 3: 参数验证失败
```json
{
  "error": {
    "message": "invalid parameter: aspect_ratio",
    "code": "invalid_request"
  }
}
```

**原因**：使用了模型不支持的参数

**解决**：
- MiniMax H3 / Video 2.0 / Wan3.0 → 使用 `aspect_ratio`
- SD2 / Seedance → 使用 `ratio`

---

## 📊 参数对照表（基于代码分析）

| 模型系列 | 接口路径 | 画幅字段 | 分辨率字段 | 参考素材字段 |
|---------|---------|---------|----------|------------|
| minimax-h3-* | /v1/video/async-generations | aspect_ratio | ❌ 固定 | image_url, image_urls, audio_url |
| video-2.* | /v1/video/async-generations | aspect_ratio | ✅ 必传 | image_urls, video_reference, audio_url |
| wan3.0-* | /v1/video/async-generations | aspect_ratio | ❌ 固定 | image_urls, audio_url |
| sd2.* | 根据渠道配置 | ratio | 可选 | referenceImages, referenceVideos |
| seedance-* | 根据渠道配置 | ratio | 可选 | referenceImages, referenceVideos |
| sora2, kling-v3, veo31, grok | /v1/video/async-generations | 根据上游 | 根据上游 | 根据上游 |

---

## 🔍 如何获取准确的模型信息

### 方法 1: 使用 Chrome DevTools 查看网络请求
1. 打开 `https://ghlink.top/pricing`
2. 按 F12 打开开发者工具
3. 切换到 Network 标签
4. 点击某个模型查看详情
5. 查看实际发送的请求参数

### 方法 2: 查询 API 文档接口
```bash
# 获取可用模型列表
curl https://ghlink.top/v1/models \
  -H "Authorization: Bearer sk-你的密钥"
```

### 方法 3: 查看数据库配置
登录管理后台，查看：
- **渠道管理** → 查看具体渠道的模型配置
- **模型管理** → 查看模型映射关系

---

## ✅ 最佳实践

1. **始终使用代码中定义的模型名称**，不要使用前端显示名称
2. **查询任务状态时**，使用 3-5 秒的轮询间隔
3. **测试新模型前**，先在管理后台查看渠道配置
4. **保存 task_id**，不要使用本地生成的 ID
5. **参考素材 URL** 必须公网可访问
6. **处理响应时**，兼容多种 URL 字段（`url`, `video_url`, `output_url` 等）

---

**文档更新时间**: 2026-09-21  
**基于代码版本**: new-api main branch  
**建议**: 使用前先在测试环境验证模型名称和接口路径
