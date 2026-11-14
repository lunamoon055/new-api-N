import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { normalizeCatalog, resolveSelectedModel } from '../src/model-catalog.js'
import { getAllModelDocs } from '../src/all-model-docs.js'
import { getCatalogHttpError, getPricingRequestHeaders } from '../src/catalog-auth.js'
import { getCodeSample, getImageOperationDocs, getModelDoc, getModelKind, getRequestBody, getRequestParams } from '../src/model-data.js'

test('V6K SD aliases keep public fields and publish separate generation and reference-second rates',()=>{
  const patch=JSON.parse(readFileSync(new URL('../../v6k-sd-models-2026-11-13.json',import.meta.url),'utf8'))
  const specs=JSON.parse(readFileSync(new URL('../../../common/v6k_video_models.json',import.meta.url),'utf8'))
  for(const [name,upstream] of Object.entries(patch.model_mapping)){
    const live=specs[upstream]
    const parameters={...live,duration_field:'duration',ratio_field:'ratio',ratios:['16:9','9:16'],image_field:'referenceImages',video_field:'referenceVideos',audio_field:'referenceAudios'}
    const model={model_name:name,video_resolution_prices:patch.option_updates['billing_setting.video_resolution_prices'][name],video_reference_prices:patch.option_updates['billing_setting.video_reference_prices'][name],mapped_endpoints:[{type:'openai-video',kind:'video',path:'/v1/videos',example_duration:5,example_resolution:'720p',parameters:live,documentation_fields:parameters}]}
    const doc=getModelDoc(model,{})
    assert.equal(getRequestBody(model,doc).model,name);assert.equal(getRequestBody(model,doc).duration,5)
    const rows=getRequestParams(doc);assert.equal(rows.find(row=>row[0]==='resolution')[2],'是');assert.match(rows.find(row=>row[0]==='referenceVideos')[3],/实际总时长另计/)
    const copied=getAllModelDocs({models:[model],endpoints:{}},'https://gateway.example')
    assert.match(copied,/USD \/ 参考视频秒/);assert.match(copied,/失败退款包含附加费/)
    if(upstream==='SD2.5'){
      assert.equal(model.video_resolution_prices['4k'],undefined)
      assert.match(rows.find(row=>row[0]==='referenceVideos')[3],/支持视频参考，上游数量上限未公布/)
    }
  }
})

test('unknown supplier image metadata retains conservative fallback and complementary editing', () => {
  for (const name of ['gpt-image-2','gpt-image-2.5-flare','gpt-image-2.5-sunburst']) {
    const model = {model_name:name,mapped_endpoints:[{type:'image-generation',kind:'image',method:'POST',path:'/v1/images/generations',parameters:{capabilities_unknown:true,max_images:0,max_videos:0,max_audios:0}}]}
    const doc = getModelDoc(model,{})
    assert.deepEqual(getRequestBody(model,doc),{model:name,prompt:'一张简洁的海边日落电影海报',n:1})
    assert.deepEqual(getRequestParams(doc).map(row=>row[0]),['model','prompt','n'])
    for (const language of ['cURL','Python','TypeScript']) {
      const sample = getCodeSample(model,doc,language)
      assert.match(sample,/images\/generations/)
      assert.doesNotMatch(sample,/output_resolution|aspect_ratio|image\[\]/)
    }
    model.mapped_endpoints.push({type:'image-edit',kind:'image',method:'POST',path:'/v1/images/edits',capability_source:'upstream_verified',parameters:{image_field:'image[]',min_images:1,max_images:1,max_videos:0,max_audios:0,unknown_fields:['max_images']}})
    const [generation,edit] = getImageOperationDocs(model,{})
    assert.equal(generation.path,'/v1/images/generations')
    assert.equal(edit.path,'/v1/images/edits')
    assert.match(edit.source,/真实调用验证/)
    assert.match(getRequestParams(edit).find(row=>row[0]==='image[]')[3],/数量上限未确认/)
    assert.doesNotMatch(getRequestParams(edit).find(row=>row[0]==='image[]')[3],/JPEG|WebP/)
    assert.match(getCodeSample(model,edit,'cURL'),/image\[\]=@reference.png/)
  }
})

test('standard GPT Image docs preserve merged references and distinguish 2.5 quality and markup', () => {
  for (const name of ['gpt-image-2','gpt-image-2.5-flare','gpt-image-2.5-sunburst']) {
    const upgraded = name.includes('2.5')
    const options = {max_outputs:10,max_references:16,size_multiple:16,max_edge:3840,min_pixels:655360,max_pixels:8294400,max_aspect_ratio:3,qualities:['auto','low','medium','high',...(upgraded?['xhigh','max']:[])],backgrounds:['auto','opaque','transparent'],output_formats:['png','jpeg','webp'],supports_markup:upgraded}
    const model = {model_name:name,mapped_endpoints:[
      {type:'image-generation',kind:'image',path:'/v1/images/generations',capability_source:'multiple_channels',parameters:{resolution_field:'resolution',resolutions:['1K','2K','4K'],ratio_field:'ratio',ratios:['1:1','16:9','5:4'],image_options:options}},
      {type:'image-edit',kind:'image',path:'/v1/images/edits',capability_source:'multiple_channels',parameters:{image_field:'image[]',min_images:1,max_images:20,max_image_size_mb:50,image_options:options}},
    ]}
    const [generation,edit] = getImageOperationDocs(model,{})
    const rows = getRequestParams(generation)
    assert.match(rows.find(row=>row[0]==='n')[3],/1–10/)
    assert.match(rows.find(row=>row[0]==='size')[3],/8294400/)
    assert.equal(rows.find(row=>row[0]==='quality')[3].includes('xhigh'),upgraded)
    const editRows=getRequestParams(edit)
    assert.equal(editRows.some(row=>row[0]==='markup'),upgraded)
    assert.match(editRows.find(row=>row[0]==='image[]')[3],/1–20.*最多 16/)
    assert.match(editRows.find(row=>row[0]==='image[]')[3],/50 MB/)
    assert.match(getRequestParams(generation).find(row=>row[0]==='ratio')[3],/5:4/)
    const copied=getAllModelDocs({models:[model],endpoints:{}},'https://gateway.example')
    assert.match(copied,/output_compression/)
    assert.match(copied,/OpenAI 标准渠道最多 16/)
    assert.match(getCodeSample(model,edit,'cURL'),/image\[\]=@reference.png/)
  }
})

test('image generation and file editing are documented together without a route choice', () => {
  const parameters = {resolutions:['1K','2K','4K'], ratios:['1:1','16:9'], resolution_field:'resolution', ratio_field:'ratio', max_images:0}
  const model = {model_name:'public-image', mapped_endpoints:[
    {type:'image-edit',kind:'image',method:'POST',path:'/v1/images/edits',parameters:{...parameters,image_field:'image[]',min_images:1,max_images:20}},
    {type:'image-generation',kind:'image',method:'POST',path:'/v1/images/generations',parameters},
  ]}
  const [generation, edit] = getImageOperationDocs(model,{})
  assert.equal(generation.path,'/v1/images/generations')
  assert.equal(edit.path,'/v1/images/edits')
  assert.equal(generation.alternatives,undefined)
  assert.equal(edit.alternatives,undefined)
  assert.equal(getRequestBody(model,generation)['image[]'],undefined)
  const field = getRequestParams(edit).find(row=>row[0]==='image[]')
  assert.equal(field[1],'file[]')
  assert.match(field[3],/1–20/)
  for (const language of ['cURL','Python','TypeScript']) {
    const sample = getCodeSample(model,edit,language)
    assert.match(sample,/reference\.png/)
    assert.match(sample,/image\[\]/)
    assert.doesNotMatch(sample,/application\/json|cdn\.example\.com/)
  }
  const copied = getAllModelDocs({models:[model],endpoints:{}},'https://gateway.example')
  assert.match(copied,/文生图 · POST/)
  assert.match(copied,/参考图编辑 · POST/)
  assert.doesNotMatch(copied,/按实际协议和使用的渠道选择/)
  const differentProtocols = {...model,mapped_endpoints:[...model.mapped_endpoints,{type:'openai',kind:'text',path:'/v1/chat/completions'}]}
  assert.deepEqual(getImageOperationDocs(differentProtocols,{}),[])
  assert.equal(getModelDoc(differentProtocols,{}).alternatives.length,3)
})

test('public documentation fields survive a provider mapping change without changing live fields', () => {
  const parameters = {duration_field:'seconds', durations:[5,10,30], ratio_field:'ratio', ratios:['16:9'], resolution_field:'resolution', resolutions:['720p'], image_field:'images', audio_field:'audios', max_images:30, max_videos:0, max_audios:10}
  const documentation_fields = {...parameters, duration_field:'duration', image_field:'referenceImages', video_field:'referenceVideos', audio_field:'referenceAudios'}
  const model = {model_name:'public-video', mapped_endpoints:[{type:'openai-video', kind:'video', path:'/v1/videos', parameters, documentation_fields}]}
  const doc = getModelDoc(model,{})
  const body = getRequestBody(model,doc)
  assert.equal(body.duration,5)
  assert.equal(body.seconds,undefined)
  assert.equal(body.model,'public-video')
  assert.ok(getRequestParams(doc).some(row=>row[0]==='referenceAudios'))
  assert.equal(model.mapped_endpoints[0].parameters.duration_field,'seconds')
  assert.match(getAllModelDocs({models:[model],endpoints:{}}),/referenceAudios/)
})

test('ZH model documentation fills unpublished ratios and retains per-model references', () => {
  const specs = JSON.parse(readFileSync(new URL('../../../common/v6k_video_models.json', import.meta.url), 'utf8'))
  for (const name of ['TTTseadanco2.5/480', 'XXseadanco2.5-720', 'dola2.5', 'wan3.M']) {
    const model = {model_name: name, mapped_endpoints: [{type: 'openai-video', kind: 'video', path: '/v1/videos', query_path: '/v1/videos/{task_id}', parameters: specs[name]}]}
    const doc = getModelDoc(model, {})
    const rows = getRequestParams(doc)
    assert.equal(rows.find(row => row[0] === 'ratio')[2], '是')
    for (const ratio of ['16:9', '9:16', '1:1', '4:3', '3:4', '2:3', '3:2', '21:9']) {
      assert.ok(rows.find(row => row[0] === 'ratio')[3].includes(ratio))
    }
    const videoRow = rows.find(row => row[0] === 'videos')
    assert.ok(videoRow)
    if (specs[name].max_videos === 0) assert.match(videoRow[3], /不支持/)
    assert.match(rows.find(row => row[0] === 'images')[3], new RegExp(`最多 ${specs[name].max_images} `))
    assert.equal(getRequestBody(model, doc).model, name)
    assert.equal(getRequestBody(model, doc).seconds, specs[name].durations[0])
    assert.equal(getRequestBody(model, doc).ratio, '16:9')
  }
})

test('unknown and explicitly unsupported reference capabilities remain distinct in page and copied docs', () => {
  const model = {model_name:'new-video',mapped_endpoints:[{
    type:'openai-video',kind:'video',path:'/v1/videos',
    capability_source:'upstream_catalog',
    documentation_fields:{duration_field:'duration',resolution_field:'resolution',image_field:'referenceImages',video_field:'referenceVideos',audio_field:'referenceAudios',max_images:30,max_videos:0,max_audios:0,unknown_fields:['max_videos']},
  }]}
  const doc = getModelDoc(model,{})
  const rows = getRequestParams(doc)
  assert.match(rows.find(row=>row[0]==='referenceImages')[3],/最多 30/)
  assert.match(rows.find(row=>row[0]==='referenceVideos')[3],/未确认/)
  assert.match(rows.find(row=>row[0]==='referenceAudios')[3],/不支持/)
  assert.match(rows.find(row=>row[0]==='resolution')[3],/未确认/)
  assert.match(doc.source,/上游模型目录/)
  const copied = getAllModelDocs({models:[model],endpoints:{}},'https://gateway.example')
  assert.match(copied,/referenceVideos.*未确认/)
  assert.match(copied,/referenceAudios.*不支持/)
  const unknown={...model,mapped_endpoints:[{...model.mapped_endpoints[0],documentation_fields:{...model.mapped_endpoints[0].documentation_fields,capabilities_unknown:true}}]}
  assert.match(getRequestParams(getModelDoc(unknown,{})).find(row=>row[0]==='referenceImages')[3],/未确认/)
})

test('video ratio fallback covers every live ID while preserving explicit ranges', () => {
  const missing = {model_name: 'future-video', mapped_endpoints: [{type: 'openai-video', kind: 'video', path: '/v1/videos', parameters: {omit_ratio: true}}]}
  const doc = getModelDoc(missing, {})
  assert.equal(doc.parameters.ratios.length, 8)
  assert.equal(getRequestBody(missing, doc).ratio, '16:9')
  const published = {...missing, mapped_endpoints: [{...missing.mapped_endpoints[0], parameters: {ratio_field:'aspect_ratio', ratios:['9:16', '1:1']}}]}
  const explicit = getModelDoc(published, {})
  assert.deepEqual(explicit.parameters.ratios, ['9:16', '1:1'])
  assert.equal(getRequestBody(published, explicit).aspect_ratio, '9:16')
  const bulk = getAllModelDocs(normalizeCatalog({success:true,data:[missing]}), 'https://gateway.example.com')
  assert.match(bulk, /21:9/)
  assert.match(bulk, /"ratio": "16:9"/)
  assert.doesNotMatch(bulk, /默认不提交/)
})

test('live video note parameters drive limits, examples and bulk copied documentation', () => {
  const model = { model_name: 'Seedance2.5:720（条）', mapped_endpoints: [{
    type: 'openai-video', kind: 'video', path: '/v1/videos', method: 'POST', query_path: '/v1/videos/{task_id}', example_duration: 99,
    parameters: { duration_field: 'duration', durations: Array.from({length:26}, (_,i)=>i+5), image_field: 'referenceImages', video_field: 'referenceVideos', audio_field: 'referenceAudios', max_images: 30, max_videos: 10, max_audios: 10 },
  }] }
  const doc = getModelDoc(model, {})
  const rows = getRequestParams(doc)
  assert.match(rows.find(row => row[0] === 'duration')[3], /5–30 秒/)
  assert.match(rows.find(row => row[0] === 'referenceImages')[3], /最多 30/)
  assert.match(rows.find(row => row[0] === 'referenceVideos')[3], /最多 10/)
  assert.match(rows.find(row => row[0] === 'referenceAudios')[3], /最多 10/)
  assert.equal(getRequestBody(model, doc).duration, 5)
  const output = getAllModelDocs(normalizeCatalog({success:true,data:[model]}), 'https://gateway.example.com')
  assert.match(output, /5–30 秒/)
  assert.match(output, /最多 30/)
  assert.match(output, /referenceVideos/)
  assert.doesNotMatch(output, /\/v1\/chat\/completions/)
})

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
