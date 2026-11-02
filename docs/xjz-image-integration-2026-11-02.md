# XJZ 原生4K图片线路接入记录

核对日期：2026-11-02。文档来源：<https://api.xjzapi.cc/docs>；结构化契约来源：请求带 `X-Xiaojuzi-Media-Contract: 1` 的公开 `/api/pricing`。完整的同步契约快照见 `integrations/xjz-image/media-contract.json`。

## 当前映射与结果

本地渠道 `2 / XJZ` 已部署并完成真实网关测试：三个模型的文生图、单参考图编辑均返回 HTTP 200 / 一张图片，六个图片 URL 均能读取为有效 PNG。原“大并发”线路的 `model_price_error` 不适用于本次新线路的验收结果。

| 公开模型 | 当前上游模型 | 文生图 | 单参考图编辑 |
| --- | --- | --- | --- |
| `gpt-image-2.5-sunburst` | `gpt-image-2.5-sunburst-原生4K` | 200，52.3 秒 | 200，13.8 秒 |
| `gpt-image-2.5-flare` | `gpt-image-2.5-flare-原生4K` | 200，17.3 秒 | 200，12.5 秒 |
| `gpt-image-2` | `gpt-image-2-原生4K` | 200，14.3 秒 | 200，23.4 秒 |

本地原有渠道 1 保持禁用；用户启用的渠道 2 保持启用。未调整用户的映射、密钥、分组、售价、优先级或权重。修正参数规则中仍匹配旧“大并发”型号的条件；三条模型元信息补充对应接口及参考图说明。

## 下游文生图

`POST /v1/images/generations`，JSON；使用本站公开模型名及本站 API Key。媒体契约头由渠道注入，下游不需要持有 XJZ 密钥。

```json
{
  "model": "gpt-image-2",
  "prompt": "白色背景上的蓝色陶瓷杯",
  "n": 1,
  "resolution": "1K",
  "ratio": "1:1",
  "response_format": "url"
}
```

- 分辨率档位：`1K / 2K / 4K`。
- 标准契约的比例：`16:9, 1:1, 21:9, 2:3, 3:2, 3:4, 4:3, 9:16`。普通目录列出更多比例，但本次采用标准契约明确公布的八种。
- 一次生成 `n=1`，读取 `data[].url`。未填写规格时本站默认 `1K / 1:1`；档位不代表保证固定的像素尺寸。
- 本站兼容旧客户端 `output_resolution`、`aspect_ratio` 别名，并移除不属于新契约的 `size`。标准字段优先。
- 文生图接口不接收参考图。错误传入参考图时返回明确错误，不能静默忽略后生成无参考图片。

## 下游参考图编辑

`POST /v1/images/edits`，`multipart/form-data`，图片以 `image[]` 文件上传。字段仍使用公开模型名、`prompt, n, resolution, ratio, quality, response_format`。支持 PNG/JPEG/WebP，本站创作页面限制每张 20 MB。

```bash
curl https://YOUR_GATEWAY/v1/images/edits \
  -H "Authorization: Bearer $NEW_API_KEY" \
  -F 'model=gpt-image-2' \
  -F 'prompt=把蓝色方块改成红色，保留白色背景' \
  -F 'n=1' \
  -F 'resolution=1K' \
  -F 'ratio=1:1' \
  -F 'response_format=url' \
  -F 'image[]=@reference.png'
```

标准契约上限是 **1–20 张参考图**。本次真实验证为每模型一张；20 张边界已做转发测试，未进行上游满上限生成。不支持 mask、视频及音频参考。

创作中心和 Playground 共用文件上传转换：无参考图时发送 JSON 文生图请求，有参考图时将所选图片转换为 multipart 编辑请求。能力来自最终映射的公开端点；文生图能力为零参考图，编辑端点能力为最多 20 张。API 页面为编辑接口提供 curl、Python、TypeScript、JavaScript 的 multipart 示例，避免误用 JSON。

## 部署与验证

- 本地执行 `docker compose -f docker-compose.dev.yml up -d --build new-api`，后端已重建启动；数据库卷保留。前端地址 `http://localhost:3001`，后端 `http://localhost:3000`。
- 真实测试经过本站 `/v1/images/generations` 和 `/v1/images/edits`，公开模型名由网关映射至新线路。测试使用短期、模型受限、有额度上限的本地令牌，已禁用并设为过期；未使用个人参考媒体。
- 第一次并发测试中，基础型号被测试令牌预扣额度不足拦截；增大测试额度后重试成功。该失败发生在本站测试额度限制，不属于上游线路故障。
- 后端回归覆盖 OpenAI adapter、model、dto、middleware、relay helper、controller、router、relay；前端覆盖参考图传输、创作参数、Playground 路由及 API 示例，另完成 TypeScript 检查与生产构建。
- 浏览器工具仍报告本地浏览器控制关闭，因此没有宣称已完成登录后页面的人工交互验收。
- 上游另外公布 `/v1/images/generations/async`、`/v1/images/edits/async` 和 `/v1/images/tasks/{task_id}`。本次接入同步文生图与编辑，没有把这些新异步路径冒充本站已实现的旧异步协议。
- 发布预检遇到 `Host key verification failed`，尚未进入远程服务器部署。不得将 Git 推送或本地成功描述为线上部署完成。

本地修改前的非敏感渠道备份：`/tmp/new-api-xjz-native-channel2-before-20261102165808.json`。测试响应、图片及汇总保存在本机 `/tmp/xjz-native-local-results/`；响应中不包含 API Key。
