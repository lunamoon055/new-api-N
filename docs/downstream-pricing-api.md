# 下游模型价格接口

`GET /v1/pricing` 返回当前下游 API Key 可访问的**全部已启用模型**的本站售价，覆盖文本、嵌入、重排序、图片、音频、视频等类型。使用网站原有 API 端口，无需额外开放网络端口。请求只读取本地价格配置，不请求上游、不创建生成任务、不扣调用额度。

## 调用

携带本站签发的 API Key，与 `/v1/models` 相同；不需要网页登录 Cookie、管理员 Access Token 或 `New-Api-User`。失效、禁用、额度耗尽的 Key 按现有 TokenAuth 规则返回鉴权失败。

```sh
export API_BASE_URL="http://127.0.0.1:3001"
export API_KEY="<YOUR_API_KEY>"

# 全部模型，美元基准价
curl "$API_BASE_URL/v1/pricing" \
  -H "Authorization: Bearer $API_KEY"

# 全部模型，人民币换算价
curl "$API_BASE_URL/v1/pricing?currency=CNY" \
  -H "Authorization: Bearer $API_KEY"

# 精确查询单个模型；模型名可包含中文、斜杠、加号等字符
curl --get "$API_BASE_URL/v1/pricing" \
  -H "Authorization: Bearer $API_KEY" \
  --data-urlencode 'currency=CNY' \
  --data-urlencode 'model=XXseadanco2.0M'
```

`currency` 可选 `USD`（默认）或 `CNY`，其他值返回 400。人民币价格使用站内 `USDExchangeRate` 换算，响应的 `exchange_rate` 表示 1 USD 对应多少响应币种；它是管理员配置值，不是实时市场汇率。站内汇率无效时返回 503。

上例为当前本地开发站点地址；部署后替换为实际网关域名。接口与网站同域，不需要另开监听服务。

## 返回字段

| 字段 | 含义 |
| --- | --- |
| `success` / `object` | 成功时为 `true` / `list` |
| `schema_version` | 当前协议版本 `1` |
| `currency` / `base_currency` / `exchange_rate` | 金额币种、内部基准币种 USD、换算率 |
| `price_scope` | `base`，下列价格尚未乘分组倍率或请求参数倍率 |
| `group_ratio` | 当前用户针对 Key 可用分组的实际倍率，包含用户组优惠 |
| `token_group` | Key 指定的分组；空字符串为继承用户组，`auto` 为自动分组 |
| `data[].model_name` | 本站公开的完整模型 ID |
| `data[].price_configured` | 是否有显式价格、倍率或有效阶梯规则；`false` 不代表免费 |
| `data[].price_unit` | 见下表 |
| `data[].model_price` | 按次或按秒的基价，单位由 `price_unit` 确定；分辨率定价时读取档位表 |
| `data[].input_price` / `output_price` | 普通 Token 定价的每百万输入/输出 Token 价格 |
| `data[].video_billing_mode` | 视频/分辨率计费模式，包含 `dynamic`、`fixed`、`tiered_seconds`、`tiered_request` |
| `data[].video_resolution_prices` | 按分辨率定价的金额表，沿用现有字段名，图片也可能使用此表 |
| `data[].billing_mode` / `billing_expr` | 比率或表达式定价及原始计费表达式 |
| `data[].expression_currency` | 有表达式时为 `USD`；表达式系数始终保留美元语义 |
| `data[].enable_groups` | 此模型对当前 Key 实际可用的分组 |
| `data[].model_ratio`、`completion_ratio`、缓存/图像/音频相关 `*_ratio` | 原始无量纲计费倍率，不随币种转换 |
| `data[].supported_endpoint_types` / `mapped_endpoints` | 模型公开调用接口及参数元数据 |

| `price_unit` | 读取方式 |
| --- | --- |
| `million_tokens` | 读取 `input_price` / `output_price`，缓存、图像、音频子类别仍按相应倍率计费 |
| `request` | 读取 `model_price`；分辨率阶梯模式读取 `video_resolution_prices` 对应档位 |
| `second` | 每输出秒基价；有档位表时读取对应分辨率 |
| `request_multiplier` | 媒体模型仅配置倍率，按网关现有适配器的请求参数规则计费，不能当作 Token 单价 |
| `expression` | 读取 `billing_expr`；实际价格随用量、档位、请求头和请求参数变化，没有单一固定价 |
| `unconfigured` | 尚无有效定价，不应当作零价或可用的报价 |

真实免费模型会同时返回 `price_configured:true` 与数值 `0`。模型目录可有最长 60 秒的缓存，价格读取使用当前进程的计费配置；配置跨节点同步仍遵循网站现有同步周期。

例如，在站内汇率为 `1`、720p 档配置为 `0.2`/秒、分组倍率为 `1` 时，人民币查询会包含：

```json
{
  "success": true,
  "object": "list",
  "currency": "CNY",
  "base_currency": "USD",
  "exchange_rate": 1,
  "price_scope": "base",
  "group_ratio": {"default": 1},
  "data": [{
    "model_name": "XXseadanco2.0M",
    "price_configured": true,
    "price_unit": "second",
    "video_billing_mode": "tiered_seconds",
    "video_resolution_prices": {"720p": 0.2},
    "enable_groups": ["default"]
  }]
}
```

该档位 5 秒基价为 ¥1。最终费用还应乘 `group_ratio`，并依照所选协议考虑图片数量、其他参数倍率和实际用量。该接口提供价格目录，不执行最终账单结算。

## 权限和缓存

只返回当前 Key 的模型限制及分组范围内有已启用渠道的公开模型，过滤禁用模型元数据。`auto` Key 按当前用户的自动分组范围返回各组价格倍率。没有权限或不存在的精确查询统一返回 `data:[]`，不泄露其他组模型；不会返回渠道密钥、内部成本或管理员配置表。

响应携带 `Cache-Control: private, no-store` 和 `Vary: Authorization`，禁止共享缓存混用不同账号报价。沿用现有 API 访问频率限制，超限返回 429。
