import { getCodeSample, getImageOperationDocs, getModelDoc, getRequestParams, getResponseSample, getVideoReferenceBillingRows } from './model-data.js'

export function getAllModelDocs(catalog, baseUrl) {
  const lines = [
    '# New API 模型接入文档', '',
    `当前网站模型：${catalog.models.length} 个`,
    `API Base URL：${baseUrl}`, '',
    '## 鉴权', '',
    '使用本网关签发的 API Key。密钥只保存在服务端，不放入浏览器代码。', '',
    '```sh',
    `export API_BASE_URL="${baseUrl}"`,
    'export API_KEY="<YOUR_API_KEY>"',
    '```', '',
    '请求头：Authorization: Bearer <API_KEY>',
    '使用 GET /v1/models 确认当前令牌可调用的完整模型 ID。',
    '以下示例根据接入文档生成，尚未进行真实上游调用验收。', '',
  ]
  for (const model of catalog.models) {
    const primaryDoc = getModelDoc(model, catalog.endpoints)
    lines.push(`## ${model.model_name}`, '')
    if (primaryDoc.parameters?.image_options) lines.push('支持文字生成与参考图编辑。标准能力与当前可用范围见下方参数表。', '')
    if (model.description) lines.push(primaryDoc.parameters?.image_options ? `管理员备注（旧渠道描述请以当前参数表为准）：${model.description}` : model.description, '')
    const videoBillingRows = getVideoReferenceBillingRows(model)
    if (videoBillingRows.length) {
      lines.push('### 分辨率与参考视频计费', '', '总费用 =（生成秒数 × 生成单价 + 参考视频实际总秒数 × 参考单价）× 当前分组倍率。参考视频时长由服务器核验，失败退款包含附加费。', '', '| 输出分辨率 | USD / 生成秒 | USD / 参考视频秒 |', '| --- | --- | --- |')
      for (const row of videoBillingRows) lines.push('| ' + row.join(' | ') + ' |')
      lines.push('')
    }
    const imageOperations = getImageOperationDocs(model, catalog.endpoints)
    const docs = imageOperations.length ? imageOperations : primaryDoc.alternatives?.length
      ? primaryDoc.alternatives.map((endpoint) => getModelDoc({...model, mapped_endpoints: [endpoint]}, catalog.endpoints))
      : [primaryDoc]
    for (const doc of docs) {
      if (docs.length > 1) lines.push(`### ${imageOperations.length ? (doc.path.endsWith('/edits') ? '参考图编辑 · ' : '文生图 · ') : ''}${doc.method} ${doc.path}`, '')
    if (!doc.verified) {
      if (doc.alternatives?.length) {
        lines.push('该模型关联多个接口，请按实际协议和使用的渠道选择：')
        for (const endpoint of doc.alternatives) lines.push(`- ${endpoint.method || 'POST'} ${endpoint.path}`)
        lines.push('暂无适用于全部接口的单一调用示例。', '')
        continue
      }
      lines.push('端点信息待核对，暂无经过文档确认的调用示例。')
      if (doc.path) lines.push(`站点声明的端点：${doc.method} ${doc.path}（需进一步核对）`)
      lines.push('请确认专用协议与参数后调用。', '')
      continue
    }
    lines.push(`接口：${doc.method} ${doc.path}`, `信息来源：${doc.source}`, '')
    if (doc.sampleSupported === false) {
      lines.push('此端点使用专用请求格式，请按照所选协议构造请求。', '')
      continue
    }
    lines.push('### 参数', '', '| 参数 | 类型 | 必填 | 说明 |', '| --- | --- | --- | --- |')
    for (const param of getRequestParams(doc)) lines.push('| ' + param.join(' | ') + ' |')
    lines.push('')
    for (const [label, fence] of [['cURL', 'sh'], ['Python', 'python'], ['TypeScript', 'typescript']]) {
      const sample = getCodeSample(model, doc, label)
      if (sample) lines.push(`### ${label}`, '', '```' + fence, sample, '```', '')
    }
    if (doc.queryPath) {
      lines.push('### 查询任务', '', `接口：GET ${doc.queryPath}`, '', '```sh',
        `curl "$API_BASE_URL${doc.queryPath.replace('{task_id}', '$TASK_ID')}" -H "Authorization: Bearer $API_KEY"`,
        '```', '', '创建请求仅表示任务已受理，保存 task_id 并查询完成或失败状态。', '')
    }
    lines.push('### 返回结构示意', '', '```json', getResponseSample(doc), '```', '')
    }
  }
  lines.push('© 2026 New API · QuantumNous')
  return lines.join('\n')
}
