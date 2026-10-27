# Suanliai 005 系列接入

核对日期：2026-10-27。公开来源：[算力 AI 文档](https://suanliai.top/docs)及该站 `/api/pricing` 的精确模型条目。

| 项目 | 配置 |
| --- | --- |
| 本站模型 | `Seedance2.5:720-S（条）` |
| 上游模型 | `005系列/sd2.5(30-10-10/5-30秒/720P)` |
| 创建 | `POST /v1/videos` |
| 查询 | `GET /v1/videos/{本站任务ID}` |
| 时长 | 5–30 秒，整数秒 |
| 清晰度 | 720p |
| 图片／视频／音频 | 最多 30／10／10 个公网 URL |
| 比例 | 上游未公布范围，按本站规则提供 16:9、9:16、1:1、4:3、3:4、2:3、3:2、21:9，默认 16:9 |

下游沿用本站模型名和固定参数：

```json
{
  "model": "Seedance2.5:720-S（条）",
  "prompt": "海边日落，镜头缓缓前移",
  "duration": 10,
  "ratio": "16:9",
  "resolution": "720p",
  "referenceImages": ["https://cdn.example.com/reference.png"],
  "referenceAudios": ["https://cdn.example.com/voice.mp3"]
}
```

后端将时长转换为 `seconds`，比例和清晰度转换为 `size`（如 `1280x720`），素材转换为 `images`、`videos`、`audios`，并替换为上游精确模型 ID。也接受上游文档中的 `seconds`、`size` 请求，冲突参数和超限素材在本站拦截。暂不提供首尾帧或文件上传，避免承诺上游未公布的能力。

模型广场和 `/docs` 优先采用后端新增的 `documentation_fields` 生成文档。已实现兼容的适配器对外固定 `duration`、`ratio`、`resolution`、`referenceImages`、`referenceVideos`、`referenceAudios`；支持范围继续跟随实际映射。原有 `parameters`、接口和请求字段保持兼容，供模型测试及已有下游使用。更换到尚未实现兼容的厂商协议仍须先完成适配；不保证任意上游可以直接互换。

公开价格目录将此型号标为按次，并列出 `model_price=5.5`；本次未据此修改本站售价或计费设置。协议、边界和任务查询经过本地模拟验证，尚未提交真实付费生成任务。
