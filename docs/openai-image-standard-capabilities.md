# GPT Image 2 / 2.5 标准能力与同名渠道合并

核对日期：2026-11-07。依据 OpenAI [图片生成指南](https://developers.openai.com/api/docs/guides/image-generation)、[生成参数](https://developers.openai.com/api/reference/resources/images/methods/generate)、[编辑参数](https://developers.openai.com/api/reference/resources/images/methods/edit)。范围仅包括 `gpt-image-2`、`gpt-image-2.5-sunburst`、`gpt-image-2.5-flare` 及其官方日期快照；不按名称前缀套用给其他模型或供应商自定义型号。

| 能力 | 标准 |
| --- | --- |
| 文生图 / 编辑 | JSON `POST /v1/images/generations`；参考图使用 `POST /v1/images/edits`，支持文件上传，标准渠道也可使用 JSON 对象引用 |
| 参考图 | 官方标准最多 16 张，PNG/JPEG/WebP；文件参考最大 50 MB，受本站整体请求限制约束 |
| 生成张数 | `n=1–10`，默认 1 |
| 质量 | 2：auto/low/medium/high；2.5 另有 xhigh/max |
| 尺寸 | auto 或 WIDTHxHEIGHT；两边为 16 的倍数，最长边 3840，比例 1:3–3:1，总像素 655360–8294400 |
| 高分辨率 | 超过 2560×1440 的规格属于官方标注的实验范围 |
| 输出 | PNG/JPEG/WebP；JPEG/WebP 压缩率 0–100，显式 0 保留 |
| 背景 | auto/opaque/transparent；透明背景需 PNG/WebP，Image 2 的透明背景属于官方预览能力 |
| 编辑扩展 | mask；2.5 另有 markup 透明 PNG 标注层，两者互斥 |
| 输入保真度 | Image 2 自动高保真，不发送 input_fidelity；不继承旧模型的可调保真度选项 |

GIF、SVG、2.6 功能、Responses 工具和图片 SSE 流式处理不在本次可调用能力声明中。

## 公共模型能力不再被单个渠道抹掉

`image_options` 保存标准参数及来源。相同公开模型名的能力合并保留比例、分辨率、参考图及高级参数；请求选路负责匹配供应商实际协议，而不是清空能力字段。

XJZ 的已接入配置明确声明 20 张参考图，MegaAi 使用官方 16 图标准。当两家渠道同时启用且对当前用户可用时，合并目录保留 20 图，官方标准字段仍注明 16。XJZ 禁用时只展示当前可用的 16 图，不启用该渠道或篡改其配置。对超过 16 张的请求只选启用的 XJZ；自定义尺寸、多张生成、透明背景、输出格式、压缩、mask/markup 等只选兼容标准字段的渠道。组合要求无法同时满足时不提交生成。

分组、优先级、权重、密钥、模型映射和售价不变。初次分配、亲和渠道和重试共用同一筛选，避免第一次选对但重试到错误协议。

## 既有输入继续可用

本站 `resolution=1K/2K/4K` 与 `ratio` 菜单保留。标准渠道将其转换为符合官方约束的 `size`；`output_resolution / aspect_ratio` 旧别名继续兼容。显式 `size` 具有优先级，不会被预设覆盖。

例如 2K/16:9 转为 2048×1152，4K/9:16 转为 2160×3840；4K 正方形受官方像素上限约束，使用 2880×2880，不能发送 4096×4096。原生 XJZ 继续使用其 resolution/ratio，不混用两家的字段。

## 验证边界

上一轮三款模型各一次的真实单图编辑验收保留。本轮新增能力依据官方标准和用户要求配置，没有再提交付费生成；不把官方 16 图标准描述为已经在 MegaAi 做过 16 图真实验收。

回归覆盖合并顺序、原生渠道保留、标准请求选路、16/17/20/21 张边界、n 与多图组合、尺寸转换及边界、压缩率 0、明确 false、JSON/文件转发、模型测试参数、文档示例及旧配置兼容。
