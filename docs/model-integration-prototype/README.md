# 模型接入指南 · New API

基于选定的第 1 张效果图制作的独立文档页面。

## 成品

- `model-integration.html`：单文件构建结果，包含品牌图片、样式和脚本。需要通过网站地址访问，才能读取实时模型列表。
- `../model-integration.html`：项目文档目录中的交付文件。
- `../../web/default/public/docs/model-integration.html`：网站 `/docs` 页使用的静态文件。
- 页面打开时读取同源 `/api/pricing` 一次，与主站一样携带登录 Cookie 和 `New-Api-User` 请求头；之后通过「刷新模型目录」按钮手动更新，切换窗口或返回页面不会触发刷新。读取失败时清空旧目录并提示重试；本地文件协议下不展示离线模型。
- 「复制全部模型对接」会复制当前目录的鉴权、接口、参数、cURL、Python、TypeScript 示例。待核对模型仅包含说明。
- 当前目录与接入文档已覆盖的调用格式分别标记。缺少明确专用端点的模型保留在目录中，并显示“端点信息待核对”。
- 渠道模型映射会由服务端解析到最终上游模型，`/api/pricing` 返回对应的公开接入路径。文档仍使用网站对外模型 ID 发请求；如果同一别名跨渠道对应不同路径，会列出各路径，不猜测单一调用示例。

## 内容维护

- `src/model-catalog.js` 和 `src/use-model-catalog.js`：目录过滤、选择状态、首次加载与手动刷新。
- `src/all-model-docs.js`：批量复制文本。
- `src/model-data.js`：精确模型 ID 对应的端点、参数与代码示例。
- `src/App.jsx`：文档内容与交互。
- `src/styles.css`：桌面与手机样式。

目录来源是站点模型广场配置，能否匿名访问取决于管理员设置。实际令牌可访问的模型仍须通过 `GET /v1/models` 确认。
示例不表示真实上游生成已完成验收。

## 构建

使用 `bun install` 安装依赖，执行 `bun run build:html` 导出单文件。
本地预览使用 `bun run dev -- --host 127.0.0.1 --port 4173 --strictPort`。

本次环境使用已安装依赖及 Codex 内置 Node 完成构建，依次执行：

```sh
node node_modules/vite/bin/vite.js build
node scripts/prepare-sites-build.mjs
node scripts/export-html.mjs
```

浏览器交互与视觉校验见 `design-qa.md`。单文件已检查资源内嵌。通过 `file://` 打开时会提示使用网站文档入口，不会展示旧模型。

© 2026 New API · QuantumNous
