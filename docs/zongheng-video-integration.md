# 纵横科技视频渠道

依据 [纵横科技 API 文档](https://cnd-coo-new.pages.dev/api-docs)（2026-10-23 读取）。本渠道只接视频。

## 管理员配置

1. 新建渠道，类型选择 **Zongheng (纵横科技)**（类型编号 59）。
2. 地址填写 `https://cnd-coo-new.pages.dev`；也可填写末尾带 `/v1` 的地址。
3. 填入已授权视频分组的上游 API Key，点击获取模型。列表只保留 `capabilities.capability_type=video` 的公开 ID，保留大小写、中文和标点。
4. 保存所需模型。支持模型映射：左侧是本站公开别名，右侧必须是上游返回的完整 ID。
5. 为本站公开模型配置价格和视频计费模式：按条使用 `fixed`，按输出秒数使用 `dynamic`，按清晰度区分价格可使用 `tiered_request` / `tiered_seconds`。没有价格的模型会被拒绝提交。

文档还提及参考视频时长和 Token 等上游计费方式，但没有给出可用于自动结算的用量响应字段。本适配器不会将上游积分直接当美元，不会猜测成本或倍率；本站使用管理员配置的价格。需计入实际参考秒数或 Token 的精确成本时，应取得上游用量协议后再接对应结算规则。

## 下游创建与查询

以下示例使用本站 API Key，不使用上游 Key。模型名、时长、清晰度与比例需替换为实际能力；提交前服务端会使用该渠道 Key 校验能力，成功读取的能力可复用 60 秒，按地址、密钥和代理分别缓存。未指定参数时使用上游公开的默认值；空选项不视为任意值均支持。

```bash
curl "$GATEWAY_BASE_URL/v1/videos" \
  -H "Authorization: Bearer $GATEWAY_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{"model":"XXseadanco2.0M","prompt":"山间日出，镜头缓慢向前","duration":10,"resolution":"720p","generate_audio":false}'
```

返回本站 `task_...` 编号表示已受理。持续查询原编号，直到 `completed` 或 `failed`：

```bash
curl "$GATEWAY_BASE_URL/v1/videos/$TASK_ID" \
  -H "Authorization: Bearer $GATEWAY_API_KEY"
curl "$GATEWAY_BASE_URL/v1/videos/$TASK_ID/content" \
  -H "Authorization: Bearer $GATEWAY_API_KEY" -o result.mp4
```

成功查询的 `metadata.url` 为结果地址；配置媒体存储时使用本站转存后的地址。内容接口支持 Range 下载。成品有效期由上游返回和存储配置决定，请及时保存。

## 参考素材

- 普通图片：`images: ["https://..."]`。
- 视频、音频：`reference_videos`、`reference_audios` URL 数组。
- 首尾帧：`start_frame`、`end_frame`，尾帧需要首帧；按上游模型说明选择支持对应模式的模型，不与普通参考图混用。
- 也兼容本站已有 `referenceImages` / `referenceVideos` / `referenceAudios`、`image_url` / `video_url` / `audio_url`、`start_image_url` / `end_image_url` 字段，发往上游时转换为上述平台字段。
- URL 必须是可公开读取的 HTTPS 直链。Base64 data URL 会先上传到上游 `/v1/media`。
- 本地文件可用 multipart 提交，模型和提示词作为普通表单字段，文件字段使用 `images`、`reference_videos`、`reference_audios`、`start_frame`、`end_frame` 或 `input_reference`。单文件不超过 25MB，最多 50 个，并遵守模型更低的引用数量限制。
- 使用 `ratio` 和 `resolution`，不要附加 `size`；`duration` 使用整数秒。`generate_audio:false` 会原样保留。

## 协议与重试

上游提交固定为 `POST /v1/videos`，轮询固定为 `GET /v1/tasks/{上游任务ID}`，不按模型名称猜协议。本站 task ID 与上游 vid ID 分开保存。

同一本站提交过程的同渠道同 Key 重试使用同一 `Idempotency-Key`（本站 task ID）、能力快照和素材上传结果。该机制不表示重复调用本站创建接口会自动合并为一个订单，也不承诺跨渠道或跨账号重试去重。查询失败时继续查询原任务，不重复提交。

读取模型目录的 GET 请求遇到 EOF、连接重置或临时 HTTP 错误时，最多尝试三次，并遵守总超时与 Retry-After；401/403、参数错误和非法响应不重试。缓存过期且无法重新读取时会停止提交，不使用界面快照或过期能力继续生成。这些重试不创建视频，也不重发生成 POST。

后台遵守 `next_poll_seconds`。上游 `processing` 为处理中，`succeeded` 为成功，只有明确的 `failed` 才进入失败退款流程；查询 HTTP 错误、未知状态或成功但缺少视频地址会留待后续重查。已取得成品的媒体转存重试沿用项目现有流程。

视频接口没有流式 `StreamOptions`，不加入流式渠道列表。

## 验证边界

已用本地现有「纵横API」渠道的密钥只读验证：2026-10-23 返回 27 个模型，其中 26 个为视频，且与本地已配置的视频清单完全一致。这 26 个模型均未公布可选比例，创作中心和模型测试页不展示或提交比例字段。内置能力快照只用于纵横渠道的界面参数；实际提交按最近 60 秒内成功读取的能力校验，过期后重新读取。

本地开发环境的渠道 4「纵横API」已从类型 1 切换为类型 59，模型清单、密钥和原价格配置均保留。当前 26 个模型没有显式配置本站价格，生成请求会被拦截；页面显示的通用回退 Token 价格不能当作已配置的视频售价。正式使用前请完成价格配置。

2026-10-23 已在重新登录后的模型测试页验证真实请求：`XXseadanco2.0M`、720p、5 秒、不提交比例，成功读取上游能力后返回「未配置价格或倍率」，未创建生成任务。另一次真实请求在三次读取尝试后仍返回 EOF，说明读取重试可缓解间歇断连，但不能保证上游每次可用。

模拟上游覆盖能力发现、精确模型映射、参数校验、素材上传、稳定重试、公开任务 ID、轮询状态与结果地址。未触发真实付费生成，成品生成和上游真实扣费尚未验证。
