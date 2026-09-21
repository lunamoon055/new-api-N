# GHLINK 模型对接文档（基于实际API数据）

**数据来源**：https://ghlink.top/api/pricing  
**获取时间**：2026-09-21  
**数据版本**：a42d372ccf0b5dd13ecf71203521f9d2

---

## 📋 实际可用模型列表（15个）

### 🎬 视频生成模型（9个）

| 模型名称 | 价格(¥/秒) | 描述 | 分组 | 接口类型 |
|---------|-----------|------|------|---------|
| **官方h3-1080p** | 0.26 | 按秒计费；9图3音频；不支持视频参考 | 全部 | openai |
| **官方h3-2k** | 0.28 | 按秒计费；9图3音频；不支持视频参考 | 全部 | openai |
| **mx-h3** | 1.5/次 | 9图/3视频/3音频/原生真人/768P/4-15s | 全部 | openai |
| **Seedance2.0-c** | 4.7/次 | 4-15s，满血933 | 全部 | openai |
| **Seedance2.0-fast-c** | 2.6/次 | 按次计费 | 全部 | openai |
| **Seedance2.0-fast2-c** | 3.0/次 | 按次计费 | 全部 | openai |
| **Seedance2.5-c** | 3.2/次 | 10图/有几率卡真人人脸/只能30s | 全部 | openai |
| **sd2.0(满血9-3-3不卡脸720P)** | 3.2/次 | 满血9-3-3/5-15秒不卡脸720P | 全部 | openai |
| **wan3.0(图片+音频参考)** | 0.28/秒 | 图片+音频参考，支持480p/720p/1080p | 全部 | openai |
| **d2.5(满血原生过脸30-10-10/480P)** | 0.55/次 | 满血原生过脸，30图10视频10音频 | 全部 | openai |

### 🖼️ 图片生成模型（6个）

| 模型名称 | 价格(¥/次) | 描述 | 分组 | 接口类型 |
|---------|-----------|------|------|---------|
| **gpt-image-2** | 0.08 | medium质量，原生1k2k4k，支持17张参考图 | 全部 | image-generation, openai |
| **gpt-image-2.5-flare** | 0.08 | medium质量，原生1k2k4k，支持16张参考图 | 全部 | image-generation, openai |
| **gpt-image-2.5-sunburst** | 0.08 | medium质量，原生1k2k4k，支持16张参考图 | 全部 | image-generation, openai |
| **nano-banana-pro** | 0.09 | - | 全部 | image-generation, openai |
| **nano-banana2** | 0.09 | - | 全部 | image-generation, openai |

---

## 🔌 API 接口路径（官方）

根据API返回的 `supported_endpoint` 字段：

### 图片生成接口
```
POST /v1/images/generations
```

### 视频生成接口（OpenAI 兼容）
```
POST /v1/chat/completions
```
或根据项目代码：
```
POST /v1/video/async-generations
GET  /v1/video/async-generations/:task_id
```

---

## 📝 各模型详细对接说明

### 1. 官方h3-1080p

**模型标识**: `官方h3-1080p`  
**价格**: ¥0.26/秒  
**计费方式**: 按秒计费

#### 功能特性
- ✅ 支持 9 张参考图片
- ✅ 支持 3 个参考音频
- ❌ 不支持视频参考
- 📐 固定分辨率：1080P

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "官方h3-1080p",
    "prompt": "一只可爱的猫咪在阳光下玩耍",
    "duration": 8,
    "aspect_ratio": "16:9"
  }'
```

#### 参数说明
```json
{
  "model": "官方h3-1080p",           // 必填，模型名称（完全一致）
  "prompt": "string",                // 必填，视频描述
  "duration": 8,                     // 视频时长（秒）
  "aspect_ratio": "16:9",            // 画幅比例
  "image_urls": ["url1", "url2"],    // 可选，最多9张参考图
  "audio_url": "url"                 // 可选，最多3个音频
}
```

---

### 2. 官方h3-2k

**模型标识**: `官方h3-2k`  
**价格**: ¥0.28/秒  
**计费方式**: 按秒计费

#### 功能特性
- ✅ 支持 9 张参考图片
- ✅ 支持 3 个参考音频
- ❌ 不支持视频参考
- 📐 固定分辨率：2K

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "官方h3-2k",
    "prompt": "科幻场景，宇航员漫步火星表面",
    "duration": 10,
    "aspect_ratio": "16:9"
  }'
```

---

### 3. mx-h3

**模型标识**: `mx-h3`  
**价格**: ¥1.5/次  
**计费方式**: 按次计费

#### 功能特性
- ✅ 支持 9 张参考图片
- ✅ 支持 3 个参考视频
- ✅ 支持 3 个参考音频
- ✅ 原生真人支持
- 📐 固定分辨率：768P
- ⏱️ 时长范围：4-15 秒

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "mx-h3",
    "prompt": "根据参考素材生成商业广告",
    "duration": 10,
    "aspect_ratio": "16:9",
    "image_urls": ["url1", "url2"],
    "video_reference": [{"url": "url3"}],
    "audio_url": "url4"
  }'
```

---

### 4. Seedance2.0-c

**模型标识**: `Seedance2.0-c`  
**价格**: ¥4.7/次  
**计费方式**: 按次计费  
**厂商**: 即梦 (Jimeng)

#### 功能特性
- ⏱️ 时长范围：4-15 秒
- 🔥 满血933版本

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seedance2.0-c",
    "prompt": "电影级别的城市夜景",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "720p"
  }'
```

**注意**: Seedance 系列使用 `ratio` 而非 `aspect_ratio`

---

### 5. Seedance2.0-fast-c

**模型标识**: `Seedance2.0-fast-c`  
**价格**: ¥2.6/次  
**计费方式**: 按次计费  
**厂商**: 即梦 (Jimeng)

#### 功能特性
- ⚡ 快速生成版本
- 💰 价格更优惠

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seedance2.0-fast-c",
    "prompt": "快速生成测试视频",
    "duration": 8,
    "ratio": "16:9"
  }'
```

---

### 6. Seedance2.0-fast2-c

**模型标识**: `Seedance2.0-fast2-c`  
**价格**: ¥3.0/次  
**计费方式**: 按次计费  
**厂商**: 即梦 (Jimeng)

#### 功能特性
- ⚡⚡ 更快速的生成版本

---

### 7. Seedance2.5-c

**模型标识**: `Seedance2.5-c`  
**价格**: ¥3.2/次  
**计费方式**: 按次计费  
**厂商**: 即梦 (Jimeng)

#### 功能特性
- ✅ 支持 10 张参考图片
- ⚠️ 有几率卡真人人脸
- ⏱️ 固定时长：30 秒

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "Seedance2.5-c",
    "prompt": "30秒长视频内容",
    "duration": 30,
    "ratio": "16:9"
  }'
```

---

### 8. sd2.0(满血9-3-3不卡脸720P)

**模型标识**: `sd2.0(满血9-3-3不卡脸720P)`  
**价格**: ¥3.2/次  
**计费方式**: 按次计费  
**厂商**: 即梦 (Jimeng)

#### 功能特性
- ✅ 支持 9 张参考图片
- ✅ 支持 3 个参考视频
- ✅ 支持 3 个参考音频
- ✅ 不卡真人人脸
- 📐 固定分辨率：720P
- ⏱️ 时长范围：5-15 秒

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "sd2.0(满血9-3-3不卡脸720P)",
    "prompt": "真人面部特写，表情自然",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "720p",
    "referenceImages": ["url1", "url2"]
  }'
```

**注意**: SD2 系列使用 `referenceImages/Videos/Audios` 字段

---

### 9. wan3.0(图片+音频参考)

**模型标识**: `wan3.0(图片+音频参考)`  
**价格**: ¥0.28/秒  
**计费方式**: 按秒计费  
**图标**: @Wan

#### 功能特性
- ✅ 支持图片参考
- ✅ 支持音频参考
- 📐 支持多种分辨率：480p/720p/1080p

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "wan3.0(图片+音频参考)",
    "prompt": "根据图片和音频生成视频",
    "duration": 10,
    "aspect_ratio": "16:9",
    "resolution": "1080p",
    "image_urls": ["url1"],
    "audio_url": "url2"
  }'
```

---

### 10. d2.5(满血原生过脸30-10-10/480P)

**模型标识**: `d2.5(满血原生过脸30-10-10/480P)`  
**价格**: ¥0.55/次  
**计费方式**: 按次计费

#### 功能特性
- ✅ 满血原生过脸
- ✅ 支持 30 张参考图片
- ✅ 支持 10 个参考视频
- ✅ 支持 10 个参考音频
- 📐 固定分辨率：480P

#### API 请求示例
```bash
curl -X POST https://ghlink.top/v1/video/async-generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "d2.5(满血原生过脸30-10-10/480P)",
    "prompt": "大量素材参考的视频生成",
    "duration": 10,
    "ratio": "16:9",
    "resolution": "480p",
    "referenceImages": ["url1", "url2", "..."],
    "referenceVideos": ["url3"],
    "referenceAudios": ["url4"]
  }'
```

---

## 🖼️ 图片生成模型

### 11-15. GPT Image 系列 & Nano Banana 系列

所有图片模型使用相同的接口：

```bash
curl -X POST https://ghlink.top/v1/images/generations \
  -H "Authorization: Bearer sk-你的密钥" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "gpt-image-2",
    "prompt": "一幅美丽的风景画",
    "size": "1024x1024",
    "n": 1
  }'
```

#### 模型对比

| 模型 | 价格 | 质量 | 参考图数量 | 分辨率支持 |
|------|------|------|-----------|----------|
| gpt-image-2 | ¥0.08 | medium | 17张 | 1k/2k/4k |
| gpt-image-2.5-flare | ¥0.08 | medium | 16张 | 1k/2k/4k |
| gpt-image-2.5-sunburst | ¥0.08 | medium | 16张 | 1k/2k/4k |
| nano-banana-pro | ¥0.09 | - | - | - |
| nano-banana2 | ¥0.09 | - | - | - |

---

## 🎯 模型名称使用规范

### ✅ 正确示例
```json
{
  "model": "官方h3-2k"  // 完全一致，包括中文和符号
}
```

```json
{
  "model": "sd2.0(满血9-3-3不卡脸720P)"  // 完整括号和中文
}
```

### ❌ 错误示例
```json
{
  "model": "官方h3-2K"  // 错误：大小写不一致
}
```

```json
{
  "model": "sd2.0"  // 错误：缺少括号部分
}
```

```json
{
  "model": "Seedance2.0-C"  // 错误：大小写错误
}
```

---

## 📊 参数字段对照表

| 模型系列 | 画幅字段 | 分辨率字段 | 参考图片 | 参考视频 | 参考音频 |
|---------|---------|----------|---------|---------|---------|
| 官方h3-* | `aspect_ratio` | ❌ 固定 | `image_urls` (9张) | ❌ | `audio_url` (3个) |
| mx-h3 | `aspect_ratio` | ❌ 固定768P | `image_urls` (9张) | `video_reference` (3个) | `audio_url` (3个) |
| Seedance* | `ratio` | 可选 | `referenceImages` | `referenceVideos` | `referenceAudios` |
| sd2.0(...) | `ratio` | `resolution` | `referenceImages` (9张) | `referenceVideos` (3个) | `referenceAudios` (3个) |
| wan3.0(...) | `aspect_ratio` | `resolution` | `image_urls` | ❌ | `audio_url` |
| d2.5(...) | `ratio` | `resolution` | `referenceImages` (30张) | `referenceVideos` (10个) | `referenceAudios` (10个) |

---

## 🔍 用户分组说明

根据API数据，所有模型都启用了以下用户分组：
- `SVIP` - SVIP用户
- `vip` - VIP用户
- `默认` - 默认用户组
- `default` - 默认组

**分组倍率**：
- `SVIP`: 1.0x
- `默认`: 1.0x

---

## ⚠️ 重要提示

1. **模型名称必须完全一致**
   - 包括中文字符
   - 包括括号和特殊符号
   - 区分大小写

2. **计费方式**
   - 按秒计费：官方h3-1080p (¥0.26/秒)、官方h3-2k (¥0.28/秒)、wan3.0 (¥0.28/秒)
   - 按次计费：其他所有模型

3. **参数字段差异**
   - 官方h3/mx-h3/wan3.0: 使用 `aspect_ratio` 和 `image_urls/audio_url`
   - Seedance/sd2.0/d2.5: 使用 `ratio` 和 `referenceImages/Videos/Audios`

4. **参考素材限制**
   - 不同模型支持的参考素材数量不同
   - 超过限制会返回错误

---

## 🚀 快速测试脚本

```bash
#!/bin/bash

API_KEY="sk-你的密钥"
BASE_URL="https://ghlink.top"

# 测试官方h3-2k
curl -X POST $BASE_URL/v1/video/async-generations \
  -H "Authorization: Bearer $API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "官方h3-2k",
    "prompt": "测试视频生成",
    "duration": 5,
    "aspect_ratio": "16:9"
  }'
```

---

**数据版本**: a42d372ccf0b5dd13ecf71203521f9d2  
**文档生成时间**: 2026-09-21  
**数据来源**: https://ghlink.top/api/pricing （实际线上API）
