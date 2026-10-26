// The live catalog comes from /api/pricing. These exact-name additions only
// cover endpoint families documented in this repository; they do not imply that
// a particular token can use a model or that upstream generation was tested.

const imageNames = new Set([
  'gpt-image-2',
  'gpt-image-2.5-flare',
  'gpt-image-2.5-sunburst',
  'nano-banana-pro',
  'nano-banana2',
])

const gatewayVideo = new Set([
  '官方h3-1080p',
  '官方h3-2k',
  'Seedance2.0-c',
  'Seedance2.0-fast-c',
  'Seedance2.0-fast2-c',
  'Seedance2.5-c',
  'wan3.0(图片+音频参考)',
])

const exactVideoDetails = {
  '特价dola2.5-900-720P-固定30s': { duration: 30, resolution: '720p', imageRequired: true },
  '特价dola-sd2.0-900-720p': { duration: 10, resolution: '720p', imageRequired: true },
  'TTP-grok': { duration: 6, resolution: '720p' },
  'gz稳定sd2.0-903-720P': { duration: 5, resolution: '720p' },
  'Xminimex-h3-2K-933': { duration: 5, resolution: '2k' },
  '特价sd2.0-720p-933-过脸': { duration: 5, resolution: '720p' },
  '特价稳sd-2.5-30s-720P可10图': { duration: 30, resolution: '720p' },
  '特价稳sd2.0-933-720p': { duration: 5, resolution: '720p' },
  'gz稳定sd2.5-720全参过脸': { duration: 5, resolution: '720p' },
  'gz稳定sd2.5-480全参过脸': { duration: 5, resolution: '480p' },
  'gz稳定sd2.5-1080全参过脸': { duration: 5, resolution: '1080p' },
  '特价sd2.0-933-过脸': { duration: 5, resolution: '720p' },
  'MiniMaxH3': { duration: 5, resolution: '2k' },
}

// These two documented Seedance aliases are video models even when an older
// catalog falls back to the channel's generic chat protocol.
const seedanceAliases = new Set(['Seedance 2.0:X', 'Seedance 2.0:X1'])
const seedanceAliasParameters = {
  durations: Array.from({ length: 11 }, (_, i) => i + 5), resolutions: ['720p'], ratios: ['16:9', '9:16', '1:1'],
  duration_field: 'duration', ratio_field: 'ratio', resolution_field: 'resolution',
  image_field: 'referenceImages', video_field: 'referenceVideos', audio_field: 'referenceAudios',
  max_images: 9, max_videos: 3, max_audios: 3,
}
function getMappedEndpoints(model) {
  return (model.mapped_endpoints || []).filter((endpoint) => endpoint?.path)
}

const likelyVideoName = /sd2|sd-2|seedance|h3|video|grok|dola|wan3|sora|veo|kling/i
const fullVideoRatios = ['16:9', '9:16', '1:1', '4:3', '3:4', '2:3', '3:2', '21:9']

function withVideoRatios(parameters) {
  if (!parameters) return parameters
  return {
    ...parameters,
    omit_ratio: false,
    ratio_field: parameters.ratio_field || 'ratio',
    ratios: parameters.ratios?.length ? parameters.ratios : fullVideoRatios,
  }
}

export function getModelKind(model) {
  const name = model.model_name || ''
  const types = model.supported_endpoint_types || []
  const mapped = getMappedEndpoints(model)
  if (mapped.length) return mapped[0].kind || 'other'
  if (imageNames.has(name) || types.includes('image-generation')) return 'image'
  if (gatewayVideo.has(name) || exactVideoDetails[name] || likelyVideoName.test(name)) return 'video'
  if (name.toLowerCase().includes('midjourney')) return 'other'
  return 'text'
}

export function getModelDoc(model, endpointMap, selectedPath) {
  const name = model.model_name || ''
  const kind = getModelKind(model)
  const mapped = getMappedEndpoints(model)
  if (mapped.length) {
    const endpoint = mapped.find((entry) => entry.path === selectedPath) || mapped[0]
    const path = endpoint.path.replace('{model}', encodeURIComponent(name))
    return {
      kind: endpoint.kind,
      type: endpoint.type,
      selectionPath: endpoint.path,
      ...(mapped.length > 1 ? { alternatives: mapped } : {}),
      method: endpoint.method || 'POST',
      path,
      queryPath: endpoint.query_path?.replace('{model}', encodeURIComponent(name)),
      source: '渠道映射对应的上游模型',
      verified: true,
      mapped: true,
      parameters: endpoint.kind === 'video' ? withVideoRatios(endpoint.parameters) : endpoint.parameters,
      sampleSupported: endpoint.type === 'image-generation' || endpoint.type === 'openai' ||
        ['openai-video', 'video-videos', 'video-async', 'video-generations', 'openai-response', 'openai-response-compact', 'anthropic', 'gemini', 'embeddings', 'jina-rerank'].includes(endpoint.type),
      details: endpoint.example_duration ? {
        duration: endpoint.example_duration,
        resolution: endpoint.example_resolution,
        imageRequired: endpoint.example_image_required,
      } : exactVideoDetails[name],
    }
  }
  if (seedanceAliases.has(name)) return {kind: 'video', path: '/v1/videos', method: 'POST', queryPath: '/v1/videos/{task_id}', verified: true, source: '模型视频协议', parameters: seedanceAliasParameters}
  if (kind === 'image' && imageNames.has(name)) {
    return { kind, method: 'POST', path: '/v1/images/generations', source: '仓库接入文档', verified: true }
  }
  if (kind === 'video' && (gatewayVideo.has(name) || exactVideoDetails[name])) {
    return {
      kind,
      method: 'POST',
      path: '/v1/videos',
      queryPath: '/v1/videos/{task_id}',
      source: '仓库接入文档',
      verified: true,
      details: exactVideoDetails[name] || { duration: 5 },
    }
  }

  const type = (model.supported_endpoint_types || []).find((item) => endpointMap[item]?.path)
  const endpoint = type ? endpointMap[type] : null
  if (endpoint && kind === 'text') {
    return {
      kind,
      method: endpoint.method || 'POST',
      path: endpoint.path.replace('{model}', encodeURIComponent(name)),
      source: '当前站点配置',
      verified: true,
    }
  }

  return {
    kind,
    method: endpoint?.method || 'POST',
    path: endpoint?.path || '',
    source: endpoint ? '当前站点配置' : '未提供',
    verified: false,
  }
}

export function getRequestBody(model, doc) {
  if (!doc.verified || doc.sampleSupported === false) return null
  if (doc.parameters && (doc.kind === 'image' || doc.kind === 'video')) {
    const p = doc.parameters
    const body = { model: model.model_name, prompt: doc.kind === 'image' ? '一张简洁的海边日落电影海报' : '海边日落时，人物缓缓走向镜头，画面稳定' }
    if (doc.kind === 'image') body.n = 1
    if (p.duration_field && p.durations?.length) body[p.duration_field] = doc.details?.duration || p.durations[0]
    if (p.ratio_field && p.ratios?.length) body[p.ratio_field] = p.ratios[0]
    if (p.resolution_field && p.resolutions?.length) body[p.resolution_field] = p.request_resolution || p.resolutions[0]
    if (p.min_images && p.image_field) body[p.image_field] = p.image_field.endsWith('_url') ? 'https://cdn.example.com/reference.jpg' : ['https://cdn.example.com/reference.jpg']
    return body
  }
  if (doc.kind === 'image') {
    const body = {
      model: model.model_name,
      prompt: '一张简洁的海边日落电影海报',
    }
    if (!doc.mapped) {
      body.aspect_ratio = '1:1'
      body.output_resolution = '1K'
      body.n = 1
    }
    return body
  }
  if (doc.kind === 'video') {
    const body = {
      model: model.model_name,
      prompt: '海边日落时，人物缓缓走向镜头，画面稳定',
    }
    if (doc.details?.duration) body.duration = doc.details.duration
    body.ratio = doc.details?.ratio || fullVideoRatios[0]
    if (doc.details?.resolution) body.resolution = doc.details.resolution
    if (doc.details?.imageRequired) {
      body.referenceImages = ['https://cdn.example.com/reference.jpg']
    }
    return body
  }
  if (doc.type === 'openai-response' || doc.type === 'openai-response-compact') return {model: model.model_name, input: '你好，请用一句话介绍自己。'}
  if (doc.type === 'gemini') return {contents: [{role: 'user', parts: [{text: '你好，请用一句话介绍自己。'}]}]}
  if (doc.type === 'embeddings') return {model: model.model_name, input: '你好'}
  if (doc.type === 'jina-rerank') return {model: model.model_name, query: '你好', documents: ['你好，世界']}
  return {
    model: model.model_name,
    ...(doc.type === 'anthropic' ? {max_tokens: 1024} : {}),
    messages: [{ role: 'user', content: '你好，请用一句话介绍自己。' }],
  }
}

export function getRequestParams(doc) {
  if (!doc.verified || doc.sampleSupported === false) return []
  if (doc.type === 'openai-response' || doc.type === 'openai-response-compact') return [['model', 'string', '是', '完整的模型 ID'], ['input', 'string / array', '是', '输入内容']]
  if (doc.type === 'gemini') return [['contents', 'array', '是', 'role 与 parts 组成的输入内容']]
  if (doc.type === 'embeddings') return [['model', 'string', '是', '完整的模型 ID'], ['input', 'string / array', '是', '待编码的文本']]
  if (doc.type === 'jina-rerank') return [['model', 'string', '是', '完整的模型 ID'], ['query', 'string', '是', '查询内容'], ['documents', 'array', '是', '候选文档']]
  if (doc.parameters) {
    const p = doc.parameters
    const rows = [['model', 'string', '是', '从当前令牌的模型列表复制完整 ID'], ['prompt', 'string', '是', '希望生成的内容描述']]
    if (p.duration_field) {
      const values = p.durations || []
      const contiguous = values.length > 1 && values.every((value, index) => index === 0 || value === values[index - 1] + 1)
      const range = contiguous ? `${values[0]}–${values.at(-1)}` : values.join('、')
      rows.push([p.duration_field, 'integer', '按模型', range ? `支持时长：${range} 秒` : '时长范围未公布，请以该模型备注和上游校验为准'])
    }
    if (p.ratio_field) rows.push([p.ratio_field, 'string', doc.kind === 'video' ? '是' : '否', p.ratios?.length ? `支持比例：${p.ratios.join('、')}` : '请显式填写比例，例如 16:9'])
    if (p.resolution_field) rows.push([p.resolution_field, 'string', '按模型', `支持分辨率：${(p.resolutions || []).join('、')}`])
    for (const [field, count, label] of [[p.image_field, p.max_images, '图片'], [p.video_field, p.max_videos, '视频'], [p.audio_field, p.max_audios, '音频']]) {
      if (field && count) rows.push([field, field.endsWith('_url') ? 'string' : 'string[]', field === p.image_field && p.min_images ? '是' : '否', `最多 ${count} 个可公开访问的${label} URL`])
    }
    return rows
  }
  if (doc.kind === 'image') {
    return [
      ['model', 'string', '是', '从当前令牌的模型列表复制完整 ID'],
      ['prompt', 'string', '是', '希望生成的图像描述'],
      ['aspect_ratio', 'string', '否', '例如 1:1、16:9；以模型能力为准'],
      ['output_resolution', 'string', '否', '例如 1K、2K、4K；以模型能力为准'],
      ['n', 'integer', '否', '生成数量；以模型详情为准'],
    ]
  }
  if (doc.kind === 'video') {
    return [
      ['model', 'string', '是', '完整的公开视频模型 ID'],
      ['prompt', 'string', '是', '视频内容、动作与镜头描述'],
      ['duration', 'integer', '按模型', '单位为秒；以该模型参数范围为准'],
      ['ratio', 'string', '是', `支持比例：${fullVideoRatios.join('、')}`],
      ['resolution', 'string', '按模型', '仅发送该模型支持的分辨率'],
      ['referenceImages', 'string[]', '按模型', '需要时传可公开访问的图片 URL'],
    ]
  }
  return [
    ['model', 'string', '是', '完整的模型 ID'],
    ['messages', 'array', '是', '按 role 顺序提供对话消息'],
    ['stream', 'boolean', '否', '按模型能力决定是否使用流式响应'],
  ]
}

export function getCodeSample(model, doc, language) {
  const body = getRequestBody(model, doc)
  if (!body) return ''
  const json = JSON.stringify(body, null, 2)
  if (language === 'Python') {
    return [
      'import os',
      'import requests',
      'import json',
      '',
      'base_url = os.environ["API_BASE_URL"]',
      'api_key = os.environ["API_KEY"]',
      'response = requests.post(',
      '    base_url + "' + doc.path + '",',
      '    headers={"Authorization": "Bearer " + api_key},',
      '    json=json.loads(' + JSON.stringify(json) + '),',
      ')',
      'response.raise_for_status()',
      'print(response.json())',
    ].join('\n')
  }
  if (language === 'TypeScript') {
    return [
      'const baseUrl = process.env.API_BASE_URL!',
      'const apiKey = process.env.API_KEY!',
      '',
      'const response = await fetch(baseUrl + "' + doc.path + '", {',
      '  method: "' + doc.method + '",',
      '  headers: {',
      '    Authorization: "Bearer " + apiKey,',
      '    "Content-Type": "application/json",',
      '  },',
      '  body: JSON.stringify(' + json.replace(/\n/g, '\n  ') + '),',
      '})',
      'if (!response.ok) throw new Error(await response.text())',
      'console.log(await response.json())',
    ].join('\n')
  }
  return [
    'curl -X ' + doc.method + ' "$API_BASE_URL' + doc.path + '" \\',
    '  -H "Authorization: Bearer $API_KEY" \\',
    '  -H "Content-Type: application/json" \\',
    "  -d '" + json + "'",
  ].join('\n')
}

export function getResponseSample(doc) {
  if (!doc.verified || doc.sampleSupported === false) return ''
  if (doc.type === 'gemini') return JSON.stringify({candidates: [{content: {role: 'model', parts: [{text: '你好！'}]}}]}, null, 2)
  if (doc.type === 'openai-response' || doc.type === 'openai-response-compact') return JSON.stringify({id: 'resp_example', output: [{type: 'message', content: [{type: 'output_text', text: '你好！'}]}]}, null, 2)
  if (doc.type === 'anthropic') return JSON.stringify({type: 'message', role: 'assistant', content: [{type: 'text', text: '你好！'}]}, null, 2)
  if (doc.type === 'embeddings') return JSON.stringify({data: [{embedding: [0.1, 0.2], index: 0}]}, null, 2)
  if (doc.type === 'jina-rerank') return JSON.stringify({results: [{index: 0, relevance_score: 0.9}]}, null, 2)
  if (doc.kind === 'image') {
    return JSON.stringify({ data: [{ url: 'https://example.com/generated-image.png' }] }, null, 2)
  }
  if (doc.kind === 'video') {
    return JSON.stringify({ task_id: 'task_xxxxxxxxx', status: 'queued', progress: 0 }, null, 2)
  }
  return JSON.stringify({ choices: [{ message: { role: 'assistant', content: '你好！' } }] }, null, 2)
}
