# MegaAi 图片 API 接入

核对日期：2026-11-07。来源：[接入文档](https://newapi.megabyai.cc/docs#images)、[公开模型目录](https://newapi.megabyai.cc/api/pricing)，并使用本地 `maga` 渠道密钥只读核对了 `GET /v1/models`。密钥未写入代码或本文档。

## 接入范围

本地渠道 `6 / maga` 使用 OpenAI 类型，地址为 `https://newapi.megabyai.cc`。以下三个现有模型均在该密钥可见模型列表中，无需改名或额外映射：

- `gpt-image-2`
- `gpt-image-2.5-flare`
- `gpt-image-2.5-sunburst`

本次保留渠道已有密钥、分组、权重、优先级、映射和本站售价。图片生成不使用聊天流式协议，无需新增 StreamOptions 声明。

## 下游请求

`POST /v1/images/generations`，JSON；使用本站 API Key 和公开模型名。请求为同步生成，读取 `data[].url` 或 `data[].b64_json`。本站现有转存及绘图日志链路继续处理生成结果。

```bash
curl "$GATEWAY_BASE_URL/v1/images/generations" \
  -H "Authorization: Bearer $GATEWAY_API_KEY" \
  -H 'Content-Type: application/json' \
  -d '{
    "model": "gpt-image-2.5-sunburst",
    "prompt": "白色背景上的蓝色陶瓷杯，柔和自然光",
    "n": 1
  }'
```

上游文档只给出 `model / prompt / n` 最小请求，没有为这三个模型公布分辨率、比例枚举、参考图字段及数量限制。本站不把其他供应商的 `output_resolution / aspect_ratio / images` 或 `resolution / ratio / image[]` 自动当作 MegaAi 能力。用户明确提交的现有通用图片字段仍按原有 OpenAI 请求处理；不注入其他渠道的默认值。

## 图片参考：真实调用已验证

2026-11-07，使用无个人信息的 512×512 PNG 测试图，向上游 `POST /v1/images/edits` 上传 `image[]` 文件；其他字段仅为 `model / prompt / n=1`。要求只把右下角紫红色方块改为绿色，保持其他图形、布局及中央 `K7Q4` 字样。

| 模型 | HTTP | 上游响应耗时 | 图片结果 |
| --- | --- | --- | --- |
| `gpt-image-2.5-sunburst` | 200 | 19.8 秒 | 保留独特图形、文字与布局，目标方块变绿 |
| `gpt-image-2.5-flare` | 200 | 21.7 秒 | 保留独特图形、文字与布局，目标方块变绿 |
| `gpt-image-2` | 200 | 31.3 秒 | 保留独特图形、文字与布局，目标方块变绿 |

三个请求均返回 `status=completed`、一张 `data[].b64_json` 图片。每模型提交一次，无自动重试，合计三次真实生成。并非只按 HTTP 成功判断，已分别查看返回图片。未核对上游最终账单；不能把 `usage.total_tokens=0` 理解为免费。

上一次根据文档缺项而暂时阻止 `maga` 编辑请求的限制已撤回；缺少文档不能证明接口不支持。当前本站文档、普通选路、渠道亲和与重试允许此编辑协议。适配器无论收到单文件 `image` 还是 `image[]`，均按实测的 `image[]` 转发，保留文件内容。

```bash
curl "$GATEWAY_BASE_URL/v1/images/edits" \
  -H "Authorization: Bearer $GATEWAY_API_KEY" \
  -F 'model=gpt-image-2.5-sunburst' \
  -F 'prompt=只把右下角方块改为绿色，保留其他内容' \
  -F 'n=1' \
  -F 'image[]=@reference.png'
```

真实出图验收仍为单张 PNG 参考，但它只代表测试覆盖范围，不再作为模型能力上限。按 OpenAI 官方模型标准配置最多 16 张参考图、1–10 张生成、自定义尺寸、质量、背景和输出格式；官方标准配置与真实上游批量验收分开记录。

## 与 XJZ 同名模型共存

公开模型 ID 保持不变。两家渠道同时启用且对用户可用时，目录保留 XJZ 的 20 图能力、1K/2K/4K 和已有比例，同时补充 MegaAi 的官方标准能力；不再把较弱测试快照覆盖到同名模型。只有 MegaAi 可用时显示官方 16 图标准，XJZ 禁用状态保持不变。

初次选路、渠道亲和与失败重试共用请求能力筛选：自定义 `size`、多张生成、背景、输出格式、压缩、mask/markup 等请求选择支持相应字段的标准渠道；17–20 张参考图走已声明 20 图的 XJZ 渠道。如果当前分组没有满足全部参数的渠道，本站直接返回无可用渠道，不会先丢弃字段后生成。

MegaAi 接收本站原有 `resolution / ratio`（及旧别名），转为合法的官方 `size=WIDTHxHEIGHT`。显式 `size` 优先，原预设字段不会重复传给上游。XJZ 的原生字段和转换保持独立。

详细标准、来源及验证边界见 [OpenAI 图片标准能力](openai-image-standard-capabilities.md)。

## 验证与范围

- 已确认渠道密钥有效、三个模型均可见；渠道密钥、分组、权重、优先级和本站售价未修改。
- 三次真实调用直接提交到 `maga` 的上游，确认供应商的单图参考能力；不把它们描述成已经过本站的付费端到端验收。
- 本地回归验证本站适配器文件内容、字段名、映射选路、亲和与重试、同名渠道能力合并、模型测试参数及文档示例。
- 原始响应、测试图及结果保存在本机 `/tmp/megabyai-reference-check/`；响应中不含 API Key。
- 本次更新本地代码与运行环境，未推送或部署线上。
