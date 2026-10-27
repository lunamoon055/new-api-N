# meaicc Wan 3.0 接入

核对日期：2026-10-27。本站模型 `Wan 3.0` 映射到 meaicc 的 `w3-c1`。

- 创建：`POST /v1/videos`
- 查询：`GET /v1/videos/{task_id}`
- 时长：5–30 秒。
- 分辨率：720p、1080p。
- 比例：16:9、9:16、1:1、4:3、3:4、21:9，与上游创作页面公布的选项一致。
- 参考素材：最多 10 张图片、5 段视频、5 段音频；支持首尾帧，计入图片总数。
- 生成结果链接有效期约 3–8 小时，获取结果后应及时保存。

```json
{
  "model": "Wan 3.0",
  "prompt": "海边日落，镜头平稳向前移动",
  "duration": 5,
  "ratio": "16:9",
  "resolution": "720p",
  "referenceImages": ["https://cdn.example.com/reference.jpg"]
}
```

网关仅在渠道地址主机为 `api.meaicc.com` 且映射后模型为 `w3-c1` 时，将请求转换成上游的 `input.prompt`、`input.media`、`parameters.duration/ratio/resolution`。素材类型分别为 `reference_image`、`reference_video`、`reference_voice`，首尾帧为 `first_frame`、`last_frame`，上游分辨率使用 `720P/1080P`。

创建响应代表任务被受理，应继续查询任务结果；上游查询返回的 `SUCCEEDED` 与 `object` 视频地址由现有任务适配器归一化。

来源：

- [上游公开模型目录](https://api.meaicc.com/pricing)：`w3-c1` 的时长、分辨率、素材上限、链接有效期。
- [上游创作页面](https://api.meaicc.com/create/)及其公开的 [模型配置](https://api.meaicc.com/create/js/config.js)、[提交实现](https://api.meaicc.com/create/js/api.js)、[查询实现](https://api.meaicc.com/create/js/polling.js)：确认请求路径、嵌套格式、素材类型与查询格式。

本次不调整本站售价，不提交真实付费视频任务。
