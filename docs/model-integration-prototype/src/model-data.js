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
  'TTP-grok': { duration: 6, resolution: '480p' },
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

const likelyVideoName = /sd2|sd-2|seedance|h3|video|grok|dola|wan3|sora|veo|kling/i

export function getModelKind(model) {
  const name = model.model_name || ''
  const types = model.supported_endpoint_types || []
  const mapped = model.mapped_endpoints || []
  if (mapped.length && mapped.every((endpoint) => endpoint.kind === mapped[0].kind)) return mapped[0].kind
  if (mapped.length) return 'other'
  if (imageNames.has(name) || types.includes('image-generation')) return 'image'
  if (gatewayVideo.has(name) || exactVideoDetails[name] || likelyVideoName.test(name)) return 'video'
  if (name.toLowerCase().includes('midjourney')) return 'other'
  return 'text'
}

export function getModelDoc(model, endpointMap) {
  const name = model.model_name || ''
  const kind = getModelKind(model)
  const mapped = (model.mapped_endpoints || []).filter((endpoint) => endpoint?.path)
  if (mapped.length > 1) {
    return { kind, method: 'POST', path: '', source: '渠道映射', verified: false, alternatives: mapped }
  }
  if (mapped.length === 1) {
    const endpoint = mapped[0]
    const path = endpoint.path.replace('{model}', encodeURIComponent(name))
    return {
      kind: endpoint.kind,
      method: endpoint.method || 'POST',
      path,
      queryPath: endpoint.query_path?.replace('{model}', encodeURIComponent(name)),
      source: '渠道映射对应的上游模型',
      verified: true,
      mapped: true,
      sampleSupported: endpoint.type === 'image-generation' || endpoint.type === 'openai' ||
        ['openai-video', 'video-videos', 'video-async', 'video-generations'].includes(endpoint.type),
      details: endpoint.example_duration ? {
        duration: endpoint.example_duration,
        resolution: endpoint.example_resolution,
        imageRequired: endpoint.example_image_required,
      } : exactVideoDetails[name],
    }
  }
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
  if (doc.kind === 'image') {
    const body = {
      model: model.model_name,
      prompt: '一张简洁的海边日落电影海报',
    }
    if (!doc.mapped) {
      body.aspect_ratio = '1:1'
      body.resolution = '1K'
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
    if (doc.details?.ratio) body.ratio = doc.details.ratio
    else if (doc.details) body.ratio = '16:9'
    if (doc.details?.resolution) body.resolution = doc.details.resolution
    if (doc.details?.imageRequired) {
      body.referenceImages = ['https://cdn.example.com/reference.jpg']
    }
    return body
  }
  return {
    model: model.model_name,
    messages: [{ role: 'user', content: '你好，请用一句话介绍自己。' }],
  }
}

export function getRequestParams(doc) {
  if (!doc.verified || doc.sampleSupported === false) return []
  if (doc.kind === 'image') {
    return [
      ['model', 'string', '是', '从当前令牌的模型列表复制完整 ID'],
      ['prompt', 'string', '是', '希望生成的图像描述'],
      ['aspect_ratio', 'string', '否', '例如 1:1、16:9；以模型能力为准'],
      ['resolution', 'string', '否', '例如 1K、2K；以模型能力为准'],
      ['n', 'integer', '否', '生成数量；以模型详情为准'],
    ]
  }
  if (doc.kind === 'video') {
    return [
      ['model', 'string', '是', '完整的公开视频模型 ID'],
      ['prompt', 'string', '是', '视频内容、动作与镜头描述'],
      ['duration', 'integer', '按模型', '单位为秒；以该模型参数范围为准'],
      ['ratio', 'string', '否', '例如 16:9、9:16'],
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
      '',
      'base_url = os.environ["API_BASE_URL"]',
      'api_key = os.environ["API_KEY"]',
      'response = requests.post(',
      '    base_url + "' + doc.path + '",',
      '    headers={"Authorization": "Bearer " + api_key},',
      '    json=' + json.replace(/\n/g, '\n    ') + ',',
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
  if (doc.kind === 'image') {
    return JSON.stringify({ data: [{ url: 'https://example.com/generated-image.png' }] }, null, 2)
  }
  if (doc.kind === 'video') {
    return JSON.stringify({ task_id: 'task_xxxxxxxxx', status: 'queued', progress: 0 }, null, 2)
  }
  return JSON.stringify({ choices: [{ message: { role: 'assistant', content: '你好！' } }] }, null, 2)
}
