import test from 'node:test'
import assert from 'node:assert/strict'
import { normalizeCatalog, resolveSelectedModel } from '../src/model-catalog.js'
import { getAllModelDocs } from '../src/all-model-docs.js'
import { getCatalogHttpError, getPricingRequestHeaders } from '../src/catalog-auth.js'
import { getCodeSample, getModelDoc, getModelKind, getRequestBody } from '../src/model-data.js'

test('pricing requests send the logged-in site user ID header', () => {
  const storage = { getItem: (key) => key === 'uid' ? '42' : null }
  assert.deepEqual(getPricingRequestHeaders(storage), { 'New-Api-User': '42' })
  assert.deepEqual(getPricingRequestHeaders({ getItem: () => null }), {})
  assert.deepEqual(getPricingRequestHeaders({ getItem: () => 'invalid\nvalue' }), {})
  assert.deepEqual(getPricingRequestHeaders({ getItem: () => { throw new Error('blocked') } }), {})
  assert.match(getCatalogHttpError(401), /鉴权失败/)
  assert.match(getCatalogHttpError(403), /无权访问/)
})

test('live catalog is authoritative, including removed, empty and duplicate entries', () => {
  const old = normalizeCatalog({ success: true, data: [{ model_name: 'gpt-image-2' }, { model_name: 'nano-banana2' }] })
  const next = normalizeCatalog({ success: true, data: [{ model_name: 'nano-banana2' }, { model_name: 'nano-banana2' }, null, { model_name: '' }] })
  assert.equal(old.models.length, 2)
  assert.deepEqual(next.models.map((model) => model.model_name), ['nano-banana2'])
  assert.equal(resolveSelectedModel(next.models, 'gpt-image-2'), 'nano-banana2')
  const empty = normalizeCatalog({ success: true, data: [] })
  assert.deepEqual(empty.models, [])
  assert.equal(resolveSelectedModel(empty.models, 'nano-banana2'), '')
  assert.throws(() => normalizeCatalog({ success: false, data: old.models }))
  assert.throws(() => normalizeCatalog({ success: true, data: null }))
})

test('bulk copy contains exactly the live models, full examples, auth, and pending protocol notes', () => {
  const catalog = normalizeCatalog({ success: true, data: [
    { model_name: '特价dola2.5-900-720P-固定30s' },
    { model_name: 'gpt-image-2' },
    { model_name: 'midjourney-默认1次4张1K' },
  ], supported_endpoint: {} })
  const output = getAllModelDocs(catalog, 'https://gateway.example.com')
  assert.match(output, /当前网站模型：3 个/)
  assert.match(output, /Authorization: Bearer <API_KEY>/)
  assert.match(output, /export API_BASE_URL="https:\/\/gateway.example.com"/)
  assert.match(output, /"duration": 30/)
  assert.match(output, /"referenceImages":/)
  assert.match(output, /GET \/v1\/videos\/\{task_id\}/)
  assert.equal((output.match(/^### cURL$/gm) || []).length, 2)
  assert.equal((output.match(/^### Python$/gm) || []).length, 2)
  assert.equal((output.match(/^### TypeScript$/gm) || []).length, 2)
  assert.match(output, /## midjourney-默认1次4张1K\n\n端点信息待核对/)
  assert.doesNotMatch(output, /nano-banana2|Seedance2.5-c/)
})

test('mapped public model inherits the upstream endpoint while requests keep the public ID', () => {
  const catalog = normalizeCatalog({ success: true, data: [
    { model_name: 'Grok', mapped_endpoints: [{
      type: 'openai-video', kind: 'video', method: 'POST', path: '/v1/videos',
      query_path: '/v1/videos/{task_id}',
      example_duration: 6, example_resolution: '480p',
    }] },
  ] })
  const model = catalog.models[0]
  assert.equal(getModelKind(model), 'video')
  const doc = getModelDoc(model, catalog.endpoints)
  assert.equal(doc.path, '/v1/videos')
  assert.equal(doc.queryPath, '/v1/videos/{task_id}')
  assert.equal(doc.verified, true)
  const sample = getCodeSample(model, doc, 'cURL')
  assert.match(sample, /\/v1\/videos/)
  assert.match(sample, /"model": "Grok"/)
  assert.match(sample, /"duration": 6/)
  assert.match(sample, /"resolution": "480p"/)
  assert.doesNotMatch(sample, /grok-imagine-video-1\.5/)
  const all = getAllModelDocs(catalog, 'https://gateway.example.com')
  assert.match(all, /接口：POST \/v1\/videos/)
  assert.match(all, /"model": "Grok"/)
})

test('multiple mapped protocols offer separate selectable examples', () => {
  const model = { model_name: 'alias', mapped_endpoints: [
    { type: 'openai-video', kind: 'video', method: 'POST', path: '/v1/videos' },
    { type: 'openai-video', kind: 'video', method: 'POST', path: '/v1/video/async-generations' },
  ] }
  const doc = getModelDoc(model, {})
  assert.equal(doc.verified, true)
  assert.equal(doc.alternatives.length, 2)
  assert.match(getCodeSample(model, doc, 'cURL'), /\/v1\/videos/)
  const selected = getModelDoc(model, {}, '/v1/video/async-generations')
  assert.match(getCodeSample(model, selected, 'cURL'), /\/v1\/video\/async-generations/)
  const all = getAllModelDocs({ models: [model], endpoints: {} }, 'https://gateway.example.com')
  assert.match(all, /POST \/v1\/videos/)
  assert.match(all, /POST \/v1\/video\/async-generations/)
  assert.equal((all.match(/^### cURL$/gm) || []).length, 2)
  const mixed = {...model, mapped_endpoints: [
    {type:'image-generation',kind:'image',path:'/v1/images/generations'},
    {type:'openai',kind:'text',path:'/v1/chat/completions'},
  ]}
  assert.equal(getModelKind(mixed),'image')
  assert.equal(getModelDoc(mixed,{},'/v1/chat/completions').kind,'text')
})

test('documented Seedance aliases use video when only generic channel metadata exists', () => {
  for (const name of ['Seedance 2.0:X', 'Seedance 2.0:X1']) {
    const model = {model_name: name, supported_endpoint_types: ['openai']}
    assert.equal(getModelKind(model), 'video')
    const doc = getModelDoc(model, {})
    assert.equal(doc.path, '/v1/videos')
    assert.equal(getRequestBody(model, doc).model, name)
    assert.equal(getRequestBody(model, doc).duration, 5)
    assert.doesNotMatch(getCodeSample(model, doc, 'cURL'), /chat\/completions|messages/)
    const explicitlyMapped = {...model, mapped_endpoints: [{type:'openai', kind:'text', path:'/v1/chat/completions'}]}
    assert.equal(getModelKind(explicitlyMapped), 'text')
    assert.match(getCodeSample(explicitlyMapped, getModelDoc(explicitlyMapped, {}), 'cURL'), /chat\/completions/)
  }
})

test('mapped native text protocols use their own payloads', () => {
  for (const [type, path, field] of [
    ['openai-response','/v1/responses','input'],
    ['gemini','/v1beta/models/{model}:generateContent','contents'],
    ['anthropic','/v1/messages','messages'],
    ['embeddings','/v1/embeddings','input'],
  ]) {
    const model = {model_name:'public-alias',mapped_endpoints:[{type, path, kind:'text',method:'POST'}]}
    const doc=getModelDoc(model,{})
    assert.ok(getRequestBody(model,doc)[field])
    assert.ok(getCodeSample(model,doc,'cURL'))
    if (type==='anthropic') assert.equal(getRequestBody(model,doc).max_tokens,1024)
  }
})

test('mapped image example does not invent upstream image options', () => {
  const model = { model_name: 'public-image', mapped_endpoints: [
    { type: 'image-generation', kind: 'image', method: 'POST', path: '/v1/images/generations' },
  ] }
  const doc = getModelDoc(model, {})
  assert.deepEqual(getRequestBody(model, doc), {
    model: 'public-image',
    prompt: '一张简洁的海边日落电影海报',
  })
})

test('examples use the same public model parameters as playground and relay', () => {
  const model = { model_name: 'dola 2.5 900', mapped_endpoints: [{
    type: 'openai-video', kind: 'video', path: '/v1/videos', method: 'POST',
    parameters: { durations: [30], resolutions: ['720p'], ratios: ['16:9'], duration_field: 'duration', ratio_field: 'ratio', resolution_field: 'resolution', image_field: 'images', min_images: 1, max_images: 9, max_videos: 0, max_audios: 0 },
  }] }
  const doc = getModelDoc(model, {})
  assert.deepEqual(getRequestBody(model, doc), {
    model: 'dola 2.5 900', prompt: '海边日落时，人物缓缓走向镜头，画面稳定', duration: 30, resolution: '720p', ratio: '16:9', images: ['https://cdn.example.com/reference.jpg'],
  })
  const mj = { model_name: 'midjourney-默认1次4张1K', mapped_endpoints: [{ type: 'image-generation', kind: 'image', path: '/v1/images/generations', parameters: { resolutions: ['1024x1024'], resolution_field: 'size', max_images: 0, max_videos: 0, max_audios: 0 } }] }
  assert.equal(getRequestBody(mj, getModelDoc(mj, {})).size, '1024x1024')
  const image = { model_name: 'gpt-image-2', mapped_endpoints: [{ type: 'image-generation', kind: 'image', path: '/v1/images/generations', parameters: { resolutions: ['1K', '2K', '4K'], ratios: ['1:1'], ratio_field: 'aspect_ratio', resolution_field: 'output_resolution', max_images: 17, max_videos: 0, max_audios: 0 } }] }
  const imageBody = getRequestBody(image, getModelDoc(image, {}))
  assert.equal(imageBody.output_resolution, '1K')
  assert.equal(imageBody.resolution, undefined)
})
