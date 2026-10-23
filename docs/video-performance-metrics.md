# 视频模型性能监测

视频监测使用 `tasks` 中已经持久化的最终状态，接入现有 `/api/perf-metrics` 和 `/api/perf-metrics/summary`，无需重新发起视频生成或回放历史计费。

## 统计口径

- 每条任务记录只贡献一个结果。`SUCCESS` 和 `FAILURE` 计入；排队、处理中及未知状态不计入。
- 图片生成、Suno 和 Midjourney 任务不混入视频监测。
- 优先使用 `properties.origin_model_name`，缺失时回退到 `upstream_model_name`，公开别名不会被上游映射名替换。
- 成功率为成功任务数除以终态任务数。平均任务耗时为提交到最终完成的平均耗时，包括排队和交付阶段；失败任务也计入。
- 完成时间优先使用 `finish_time`。旧记录缺失时使用 `updated_at`，再回退到 `submit_time`。按完成时间判断查询窗口并归入时间桶。
- 以分组隔离聚合。概览汇总按任务数加权，并在过滤无效分组后重新计算，不暴露请求量、提示词或上游凭证。

该统计直接读取任务记录，没有额外的采样写入，因此轮询、后台完成、超时处理、媒体交付及重复查询不会产生额外计数。历史记录在查询窗口内立即可见；被删除的任务记录不会继续贡献视频统计。

## 页面与接口

- 视频详情显示平均任务耗时和成功率，每 30 秒刷新一次。
- 视频性能趋势使用任务耗时；视频页面不展示基于文本 token 的 TPS、TTFT。
- 文本模型继续使用原有 relay 采样和展示方式。
- 默认查询最近 24 小时，`hours` 最大支持 720 小时。没有终态样本时显示空数据。
- 单模型响应新增可选 `data.summary`，提供加权后的 `avg_latency_ms`、`success_rate`、`avg_tps`，原有字段继续保留。
- `data.summary_series` 提供按任务数加权的整体趋势，避免对不同流量分组简单平均；视频成功率图表覆盖 0%–100%，不沿用文本可用率的 95%–100% 刻度。

## 验证

`go test ./model ./pkg/perf_metrics ./controller ./service` 覆盖任务终态、模型别名、分组过滤、重复查询、轮询状态推进及文本采样回归。

外部数据库兼容性测试为 `TestVideoTaskPerfMetricsAcrossConfiguredDialect`，通过 `PERF_METRICS_TEST_DIALECT` 选择 `mysql` 或 `postgres`，连接分别由 `PERF_METRICS_MYSQL_DSN`、`PERF_METRICS_POSTGRES_DSN` 提供。仅对独立测试数据库运行；该测试会创建任务表和测试记录。
