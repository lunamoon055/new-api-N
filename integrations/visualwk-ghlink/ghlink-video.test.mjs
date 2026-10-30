import { test } from 'node:test';
import assert from 'node:assert/strict';
import createPlugin, { api, catalog, buildPayload, pollTask, priceLabel, connectedResources, prepareReferences, uploadReference, imageResults, resultNodeOps, mentionAt, mentionOptions, insertMention, compileMentions, buildImageRequest, imageRequestResults, readResultFile } from './ghlink-video.mjs';

test('result download reads the original bytes without credentials and preserves the actual file type', async () => {
    const bytes = Uint8Array.of(1, 2, 3, 4);
    const file = await readResultFile('https://media.example/result.webp', 'image', { fetcher: async (url, init) => {
        assert.equal(url, 'https://media.example/result.webp');
        assert.equal(init.credentials, 'omit'); assert.equal(init.headers, undefined);
        assert.equal(init.redirect, 'error'); assert.equal(init.referrerPolicy, 'no-referrer');
        return new Response(bytes, { headers: { 'Content-Type': 'image/webp' } });
    } });
    assert.equal(file.filename, 'ghlink-image.webp');
    assert.deepEqual(new Uint8Array(await file.blob.arrayBuffer()), bytes);
    const video = await readResultFile('blob:https://canvas.example/fixture', 'video', { fetcher: async () => new Response(bytes, { headers: { 'Content-Type': 'video/webm' } }) });
    assert.equal(video.filename, 'ghlink-video.webm');
});

test('result download rejects missing files, HTML, empty content and unsafe addresses instead of navigating', async () => {
    const url = 'https://media.example/result.png';
    for (const [response, expected] of [
        [new Response('missing', { status: 404 }), /HTTP 404/],
        [new Response('<html>not an image</html>', { headers: { 'Content-Type': 'text/html' } }), /不是有效媒体/],
        [new Response('', { headers: { 'Content-Type': 'image/png' } }), /不是有效媒体/],
    ]) await assert.rejects(readResultFile(url, 'image', { fetcher: async () => response }), expected);
    for (const bad of ['javascript:alert(1)', 'https://secret@media.example/image.png', 'data:text/html,hello']) {
        await assert.rejects(readResultFile(bad, 'image', { fetcher: () => assert.fail('must not fetch') }), /无效/);
    }
    await assert.rejects(readResultFile(url, 'image', { fetcher: async () => { throw new TypeError('CORS blocked'); } }), /CORS blocked/);
});

test('base64 results can download directly and cancellation reaches the media fetch', async () => {
    const file = await readResultFile('data:image/png;base64,aGVsbG8=');
    assert.equal(await file.blob.text(), 'hello');
    const controller = new AbortController(); controller.abort();
    await assert.rejects(readResultFile('https://media.example/result.png', 'image', { signal: controller.signal, fetcher: async (_url, init) => {
        init.signal.throwIfAborted(); assert.fail('request should be cancelled');
    } }), { name: 'AbortError' });
});

test('completed result exposes in-site preview and download buttons without external navigation links', () => {
    const React = { createElement: (type, props, ...children) => ({ type, props: props || {}, children: children.flat(Infinity) }), useState: initial => [typeof initial === 'function' ? initial() : initial, () => {}], useRef: initial => ({ current: initial }), useEffect: () => {} };
    const plugin = createPlugin({ React });
    const definition = plugin.nodes.find(node => node.type === 'ghlink-video:result');
    const tree = definition.Content({ ctx: { node: { id: 'result', metadata: { ghMode: 'image', ghResultURL: 'https://media.example/result.png', ghProgress: { phase: 'completed' } } }, theme: { node: {}, toolbar: {} } } });
    const elements = [];
    const visit = element => { if (element && typeof element === 'object') { elements.push(element); (element.children || []).forEach(visit); } };
    visit(tree);
    assert.ok(elements.some(e => e.type === 'button' && e.children[0] === '查看大图'));
    assert.ok(elements.some(e => e.type === 'button' && e.children[0] === '下载图片'));
    assert.equal(elements.filter(e => e.type === 'a' || e.props.target === '_blank').length, 0);
});

const model = { id: 'Seedance2.5:720（条）', price_configured: true, parameters: { durations: [5, 15, 30], ratios: ['16:9'], resolutions: ['720p'], duration_field: 'seconds', image_field: 'images', audio_field: 'audios', max_images: 2, max_audios: 1, max_videos: 0 } };
const values = { prompt: '日出', seconds: '30', ratio: '16:9', resolution: '720p' };

test('authorized catalog keeps exact aliases and excludes other endpoints', () => {
    const pricing = { currency: 'CNY', price_scope: 'base', data: [
        { model_name: model.id, mapped_endpoints: [{ kind: 'video', path: '/v1/videos', parameters: model.parameters }] },
        { model_name: 'private', supported_endpoint_types: ['openai-video'] },
        { model_name: 'image', mapped_endpoints: [{ kind: 'image', path: '/v1/images/generations' }] },
        { model_name: 'kling', supported_endpoint_types: ['openai-video'], mapped_endpoints: [{ kind: 'video', path: '/kling/v1/videos/text2video' }] },
    ] };
    const result = catalog({ data: [model.id, 'image', 'kling'].map(id => ({ id })) }, pricing);
    assert.deepEqual(result.map(x => x.id), [model.id]);
    assert.equal(result[0].parameters.duration_field, 'seconds');
});

test('uses documented fields, exact model ID and durations without hardcoding five seconds', () => {
    assert.deepEqual(buildPayload(model, { ...values, audios: 'https://cdn.example/a.mp3' }), {
        model: model.id, prompt: '日出', seconds: 30, ratio: '16:9', resolution: '720p', audios: ['https://cdn.example/a.mp3'],
    });
    assert.equal(buildPayload({ ...model, parameters: { duration_field: 'duration' } }, values).duration, 30);
});

test('validates media limits including explicit zero and image prerequisites', () => {
    assert.throws(() => buildPayload(model, { ...values, videos: 'https://cdn.example/v.mp4' }), /最多支持 0/);
    assert.throws(() => buildPayload(model, { ...values, images: 'data:image/png;base64,AA==' }), /HTTP/);
    assert.throws(() => buildPayload(model, { ...values, seconds: '20' }), /时长/);
    assert.throws(() => buildPayload({ ...model, parameters: { ...model.parameters, audio_needs_image: true } }, { ...values, audios: 'https://cdn.example/a.mp3' }), /参考图/);
    assert.throws(() => buildPayload({ ...model, parameters: { duration_field: 'model' } }, values), /字段暂不支持/);
});

test('maps documented singular reference fields and resolution size without converting them into arrays', () => {
    const special = { ...model, parameters: { image_field: 'image_url', resolution_field: 'size', request_resolution: '720p' } };
    const payload = buildPayload(special, { ...values, images: 'https://cdn.example/a.png' });
    assert.equal(payload.image_url, 'https://cdn.example/a.png');
    assert.equal(payload.size, '720p');
    assert.throws(() => buildPayload(special, { ...values, images: 'https://cdn.example/a.png\nhttps://cdn.example/b.png' }), /只接受一个/);
});

test('zero price differs from unconfigured price; base price is explicitly labelled', () => {
    assert.match(priceLabel({ price_configured: true, price_unit: 'request', model_price: 0, currency: 'CNY', price_scope: 'base' }), /CNY 0.*基础价/);
    assert.match(priceLabel({ price_configured: false }), /尚未配置/);
    assert.throws(() => buildPayload({ ...model, price_configured: false }, values), /价格/);
});

test('creation makes one POST, forbids redirects and sends auth only to fixed GH origin', async () => {
    let calls = 0;
    await assert.rejects(api('test-secret', '/v1/videos', { body: { model: model.id }, fetcher: async (url, init) => {
        calls++;
        assert.equal(url, 'https://ghlink.top/v1/videos');
        assert.equal(init.method, 'POST');
        assert.equal(init.redirect, 'error');
        assert.equal(init.credentials, 'omit');
        throw new TypeError('network interrupted');
    } }), /network interrupted/);
    assert.equal(calls, 1);
    await assert.rejects(api('test-secret', 'https://other.example/v1/videos'), /不支持/);
});

test('API errors redact credentials and preserve Retry-After for polling', async () => {
    await assert.rejects(api('test-secret', '/v1/videos/id', { fetcher: async () => new Response(JSON.stringify({ error: { message: 'test-secret rejected' } }), { status: 429, headers: { 'Retry-After': '90' } }) }), error => {
        assert.equal(error.status, 429); assert.equal(error.retryMs, 90000); assert.doesNotMatch(error.message, /test-secret/); return true;
    });
});

test('unknown status remains pending; transient errors back off and never create another task', async () => {
    const waits = []; let count = 0;
    const result = await pollTask('test', 'id/中文', new AbortController().signal, () => {}, {
        wait: async ms => waits.push(ms),
        call: async (_key, path) => {
            assert.equal(path, '/v1/videos/id%2F%E4%B8%AD%E6%96%87');
            count++;
            if (count === 1) throw Object.assign(new Error('busy'), { status: 429, retryMs: 90000 });
            return count === 2 ? { status: 'unknown' } : { data: { status: 'completed' } };
        },
    });
    assert.equal(result.status, 'completed');
    assert.deepEqual(waits, [90000, 20000]);
});

test('auth errors stop polling and explicit failures are terminal', async () => {
    await assert.rejects(pollTask('test', 'id', new AbortController().signal, () => {}, { call: async () => { throw Object.assign(new Error('auth'), { status: 401 }); } }), /auth/);
    await assert.rejects(pollTask('test', 'id', new AbortController().signal, () => {}, { call: async () => ({ status: 'FAILURE', fail_reason: 'generation failed' }) }), error => error.terminal === true);
});

test('content endpoint rejects HTML errors and plugin exports a host-compatible node', async () => {
    await assert.rejects(api('test', '/v1/videos/id/content', { binary: true, fetcher: async () => new Response('<html>error</html>', { headers: { 'Content-Type': 'text/html' } }) }), /未返回视频/);
    const plugin = createPlugin({ React: { createElement() {} } });
    assert.equal(plugin.id, 'ghlink-video');
    assert.equal(plugin.nodes[0].type, 'ghlink-video:generator');
    assert.equal(plugin.nodes[0].resource({ metadata: { ghResultURL: 'javascript:alert(1)' } }), null);
});

test('connected resources select the visible image, local audio, public video and text in connection order', () => {
    const nodes = [
        { id: 'i', type: 'image', title: '图片', metadata: { primaryImageId: 'second', images: [{ id: 'first', content: 'https://cdn.example/one.png', status: 'success' }, { id: 'second', content: 'blob:old', storageKey: 'image:2', mimeType: 'image/png', status: 'success' }] } },
        { id: 'a', type: 'audio', title: '28.mp3', metadata: { content: 'blob:expired', storageKey: 'audio:1', mimeType: 'audio/mpeg' } },
        { id: 'v', type: 'video', title: '视频', metadata: { content: 'blob:expired', remoteUrl: 'https://cdn.example/v.mp4' } },
        { id: 't', type: 'text', metadata: { content: '文本提示词' } },
        { id: 'config', type: 'config', metadata: { apiKey: 'never-read' } },
    ];
    const result = connectedResources([...nodes, nodes[1]]);
    assert.deepEqual(result.map(x => x.kind), ['image', 'audio', 'video', 'text']);
    assert.equal(result[0].storageKey, 'image:2');
    assert.equal(result[2].source, 'https://cdn.example/v.mp4');
    assert.doesNotMatch(JSON.stringify(result), /never-read/);
});

test('local audio resolves by its exact storage key, uploads once and merges with manual links', async () => {
    const items = connectedResources([{ id: 'a', type: 'audio', title: '28.mp3', metadata: { content: 'blob:expired', storageKey: 'audio:1' } }]);
    let uploads = 0; const readKeys = []; const cache = new Map();
    const options = { cache, read: async (store, key) => { readKeys.push([store, key]); return store === 'media_files' ? new Blob(['audio'], { type: 'audio/mpeg' }) : null; }, upload: async (blob, item) => { uploads++; assert.equal(blob.type, 'audio/mpeg'); assert.equal(item.title, '28.mp3'); return 'https://cdn.example/a.mp3'; } };
    for (let i = 0; i < 2; i++) {
        const refs = await prepareReferences(items, { audios: 'https://cdn.example/a.mp3\nhttps://cdn.example/b.mp3' }, new AbortController().signal, options);
        assert.deepEqual(refs.audios, ['https://cdn.example/a.mp3', 'https://cdn.example/b.mp3']);
        assert.deepEqual(buildPayload(model, { ...values, audios: refs.audios.slice(0, 1) }).audios, ['https://cdn.example/a.mp3']);
    }
    assert.equal(uploads, 1);
    assert.deepEqual(readKeys, [['public_media_urls', 'audio:1'], ['media_files', 'audio:1']]);
});

test('existing public URLs bypass uploads; missing connected files fail instead of disappearing', async () => {
    const items = connectedResources([{ id: 'a', type: 'audio', metadata: { storageKey: 'audio:1' } }]);
    const refs = await prepareReferences(items, {}, new AbortController().signal, { read: async () => ({ url: 'https://cdn.example/a.mp3' }), upload: () => assert.fail('unexpected upload') });
    assert.deepEqual(refs.audios, ['https://cdn.example/a.mp3']);
    await assert.rejects(prepareReferences(items, {}, new AbortController().signal, { read: async () => null }), /无法读取连接素材/);
});

test('material count checks include all merged references and unknown capabilities do not invent zero limits', () => {
    assert.throws(() => buildPayload(model, { ...values, audios: ['https://cdn.example/a.mp3', 'https://cdn.example/b.mp3'] }), /合计 2/);
    const unknown = { ...model, parameters: { capabilities_unknown: true, max_audios: 0 } };
    assert.equal(buildPayload(unknown, { ...values, audios: ['https://cdn.example/a.mp3'] }).audios.length, 1);
});

test('uploads use the configured same-origin endpoint without GH credentials', async () => {
    let init;
    const output = await uploadReference(new Blob(['audio'], { type: 'audio/mpeg' }), { title: '28.mp3' }, new AbortController().signal, {
        origin: 'https://unline.visualwk.com', runtimeConfig: { MEDIA_UPLOAD_ENDPOINT: '/api/media/upload', MEDIA_UPLOAD_BASE_URL: 'https://media.ghlink.top' },
        fetcher: async (url, options) => { assert.equal(new URL(url).origin, 'https://unline.visualwk.com'); init = options; return new Response(JSON.stringify([{ src: '/file/a.mp3' }])); },
    });
    assert.equal(output, 'https://media.ghlink.top/file/a.mp3');
    assert.equal(init.headers, undefined); assert.equal(init.redirect, 'error');
    assert.equal(init.body.get('file').name, '28.mp3');
    await assert.rejects(uploadReference(new Blob(['a']), {}, new AbortController().signal, { origin: 'https://unline.visualwk.com', runtimeConfig: { MEDIA_UPLOAD_ENDPOINT: 'https://other.example/upload' } }), /同源/);
});

test('cancelled or oversized local media never reaches the upload service', async () => {
    const item = connectedResources([{ id: 'a', type: 'audio', metadata: { storageKey: 'audio:1' } }]);
    const options = { parameters: { max_audio_size_mb: 1 }, read: async store => store === 'media_files' ? new Blob([new Uint8Array(2 * 1024 * 1024)], { type: 'audio/mpeg' }) : null, upload: () => assert.fail('must not upload') };
    await assert.rejects(prepareReferences(item, {}, new AbortController().signal, options), /超过模型/);
    await assert.rejects(prepareReferences(item, {}, AbortSignal.abort(), options), { name: 'AbortError' });
});

test('image models use declared image reference and resolution fields, with no video duration', () => {
    const price = { data: [{ model_name: 'gpt-image-2', mapped_endpoints: [{ kind: 'image', path: '/v1/images/generations', parameters: { image_field: 'images', ratio_field: 'aspect_ratio', resolution_field: 'output_resolution', max_images: 2 } }] }] };
    const [image] = catalog({ data: [{ id: 'gpt-image-2' }] }, price, 'image');
    const payload = buildPayload(image, { prompt: '风景', resolution: '1K', ratio: '16:9', images: ['https://cdn.example/a.png'] });
    assert.deepEqual(payload, { model: 'gpt-image-2', prompt: '风景', n: 1, output_resolution: '1K', aspect_ratio: '16:9', images: ['https://cdn.example/a.png'] });
    assert.throws(() => buildPayload({ ...image, parameters: {} }, { prompt: 'test', images: ['https://cdn.example/a.png'] }), /未声明/);
});

test('image responses accept URL and base64 output, but reject task-only or malformed responses', () => {
    assert.deepEqual(imageResults({ data: [{ url: 'https://cdn.example/img.png' }] }), ['https://cdn.example/img.png']);
    assert.deepEqual(imageResults({ data: [{ b64_json: 'aGVsbG8=' }] }), ['data:image/png;base64,aGVsbG8=']);
    assert.throws(() => imageResults({ task_id: 'async-without-an-image' }), /没有返回图片/);
});


test('generation creates a separate connected canvas result and avoids existing nodes', () => {
    const source = { id: 'source', position: { x: 50, y: 100 }, width: 440, height: 760 };
    const first = resultNodeOps(source, [source], 'result1', 'video');
    assert.deepEqual(first[0].position, { x: 570, y: 100 });
    assert.equal(first[0].nodeType, 'ghlink-video:result');
    assert.deepEqual(first[1], { type: 'connect_nodes', fromNodeId: 'source', toNodeId: 'result1' });
    const second = resultNodeOps(source, [source, { ...first[0], type: first[0].nodeType }], 'result2', 'image');
    assert.deepEqual(second[0].position, { x: 570, y: 440 });
    assert.equal(second[0].metadata.ghMode, 'image');
    assert.ok(second.every(op => op.type !== 'delete_node'));
});

test('completed result nodes can be consumed as upstream media', () => {
    const items = connectedResources([{ id: 'result', type: 'ghlink-video:result', metadata: { ghMode: 'video', ghBlobKey: 'result:1' } }]);
    assert.equal(items[0].kind, 'video');
    assert.equal(items[0].pluginBlobKey, 'result:1');
    const plugin = createPlugin({ React: { createElement() {} } });
    const result = plugin.nodes.find(n => n.type === 'ghlink-video:result');
    assert.equal(result.showInCreateMenu, false);
    assert.deepEqual(result.resource({ metadata: { ghMode: 'image', ghResultURL: 'https://cdn.example/a.png' } }), { kind: 'image', url: 'https://cdn.example/a.png' });
});

const mentionItems = [
    { id: 'person', identity: 'image:person', kind: 'image', title: '同名图片.png' },
    { id: 'scene', identity: 'image:scene', kind: 'image', title: '同名图片.png' },
    { id: 'music', identity: 'audio:music', kind: 'audio', title: '28.mp3' },
    { id: 'clip', identity: 'video:clip', kind: 'video', title: '动作.mp4' },
];

test('@ detection supports Chinese text, mid-prompt cursors and full-width input without matching email', () => {
    assert.deepEqual(mentionAt('参考@人走进房间', 5), { start: 2, end: 5, query: '人走' });
    assert.deepEqual(mentionAt('使用＠', 3), { start: 2, end: 3, query: '' });
    assert.equal(mentionAt('hello@example.com', 17), null);
    assert.equal(mentionAt('参考@图片1 已选择', 10), null);
});

test('selecting an @ option replaces only the typed query and saves a node binding', () => {
    const options = mentionOptions(mentionItems);
    assert.deepEqual(options.map(x => x.token), ['@图片1', '@图片2', '@音频1', '@视频1']);
    const result = insertMention('用@同名作为人物', { start: 1, end: 4 }, options[1]);
    assert.equal(result.prompt, '用@图片2 作为人物');
    assert.equal(result.cursor, 6);
    assert.equal(result.bindings[0].nodeId, 'scene');
    const repeated = insertMention(result.prompt, { start: result.prompt.length, end: result.prompt.length }, options[1], result.bindings);
    assert.equal(repeated.bindings.length, 1);
});

test('same-named media keep stable tokens when reordered; removed labels cannot bind a different node', () => {
    const options = mentionOptions(mentionItems);
    const bindings = options.slice(0, 2).map(x => insertMention('', { start: 0, end: 0 }, x).bindings[0]);
    const reordered = mentionOptions([mentionItems[1], mentionItems[0]], bindings);
    assert.deepEqual(reordered.map(x => x.token), ['@图片2', '@图片1']);
    const newcomer = mentionOptions([{ id: 'new', identity: 'image:new', kind: 'image', title: '同名图片.png' }], bindings);
    assert.equal(newcomer[0].token, '@图片3');
});

test('prompt references use final deduplicated request order, regardless of mention order or repeated mentions', () => {
    const options = mentionOptions(mentionItems);
    const bindings = options.map(x => insertMention('', { start: 0, end: 0 }, x).bindings[0]);
    const refs = { images: ['https://cdn/scene', 'https://cdn/person'], audios: ['https://cdn/music'], videos: ['https://cdn/clip'] };
    const urls = new Map([['person', refs.images[1]], ['scene', refs.images[0]], ['music', refs.audios[0]], ['clip', refs.videos[0]]]);
    assert.equal(compileMentions('@图片1 的人物进入 @图片2，保持 @图片1 外观', bindings, mentionItems, refs, urls), '@图2 的人物进入 @图1，保持 @图2 外观');
    const multimedia = compileMentions('动作参考 @视频1，音乐用 @音频1 @音频1', bindings, mentionItems, refs, urls);
    assert.match(multimedia, /@视频1 对应上传的第 1 个参考视频/);
    assert.equal(multimedia.match(/对应上传的第 1 个参考音频/g).length, 1);
    urls.set('person', refs.images[0]);
    assert.equal(compileMentions('@图片1 @图片2', bindings, mentionItems, { ...refs, images: [refs.images[0]] }, urls), '@图1 @图1');
});

test('disconnected bound media and nonexistent numeric references fail instead of silently pointing elsewhere', () => {
    const option = mentionOptions(mentionItems)[0];
    const { bindings } = insertMention('', { start: 0, end: 0 }, option);
    const refs = { images: ['https://cdn/scene'], audios: [], videos: [] };
    assert.throws(() => compileMentions('@图片1', bindings, [mentionItems[1]], refs, new Map([['scene', refs.images[0]]])), /已断开/);
    assert.throws(() => compileMentions('@图2', [], mentionItems, refs, new Map()), /没有对应/);
    assert.equal(compileMentions('不再使用素材', bindings, [], refs, new Map()), '不再使用素材');
    assert.equal(compileMentions('手动引用 @图1', [], [], refs, new Map()), '手动引用 @图1');
});

test('reference preparation exposes each resolved URL before final deduplication for mention binding', async () => {
    const items = mentionItems.slice(0, 2).map(item => ({ ...item, source: 'https://cdn/shared.png' }));
    const resolved = new Map();
    const refs = await prepareReferences(items, {}, new AbortController().signal, { onResolved: (item, url) => resolved.set(item.id, url) });
    assert.equal(resolved.size, 2);
    assert.deepEqual(refs.images, ['https://cdn/shared.png']);
});

const dualImageModel = { id: 'gpt-image-2.5-sunburst', kind: 'image', supported_endpoint_types: ['image-generation', 'openai'], parameters: { image_field: 'images', max_images: 16, ratio_field: 'aspect_ratio', resolution_field: 'output_resolution' } };
const imageReferenceValues = { prompt: '保持 @图1 的人物衣服', images: ['https://cdn.example/ref.png'], ratio: '4:3', resolution: '1K' };

test('sunburst default request matches the successful URL-reference option without a special blocker', () => {
    assert.deepEqual(buildImageRequest(dualImageModel, imageReferenceValues), buildImageRequest(dualImageModel, imageReferenceValues, 'generations'));
    assert.equal(buildImageRequest(dualImageModel, { prompt: '日落' }).path, '/v1/images/generations');
    for (const id of ['gpt-image-2.5-flare', 'gpt-image-2', 'new-image-model']) {
        const next = { ...dualImageModel, id };
        assert.deepEqual(buildImageRequest(next, imageReferenceValues).body, buildPayload(next, imageReferenceValues));
    }
    const request = buildImageRequest(dualImageModel, imageReferenceValues, 'generations');
    assert.deepEqual(request.body.images, imageReferenceValues.images);
    assert.equal(request.body.aspect_ratio, '4:3');
    assert.equal(request.body.messages, undefined);
});

test('chat references are image content on the chat endpoint, never messages appended to images/generations', () => {
    const request = buildImageRequest(dualImageModel, imageReferenceValues, 'chat');
    assert.equal(request.path, '/v1/chat/completions');
    assert.deepEqual(Object.keys(request.body).sort(), ['messages', 'model', 'stream']);
    assert.equal(request.body.stream, false);
    const content = request.body.messages[0].content;
    assert.match(content[0].text, /4:3/);
    assert.match(content[0].text, /1K/);
    assert.deepEqual(content[2], { type: 'image_url', image_url: { url: imageReferenceValues.images[0] } });
    assert.equal(request.referenceCount, 1);
    assert.throws(() => buildImageRequest({ ...dualImageModel, supported_endpoint_types: ['image-generation'] }, imageReferenceValues, 'chat'), /未声明对话接口/);
    assert.throws(() => buildImageRequest(dualImageModel, { ...imageReferenceValues, audios: ['https://cdn.example/a.mp3'] }, 'chat'), /未声明此类参考素材/);
    assert.throws(() => buildImageRequest(dualImageModel, imageReferenceValues, 'guess-edits'), /不支持/);
});

test('chat result parsing accepts generated image parts and markdown, but rejects text, echoed inputs and unsafe URLs', () => {
    const chat = content => ({ choices: [{ message: { content } }] });
    const path = '/v1/chat/completions';
    assert.deepEqual(imageRequestResults(chat('![结果](https://cdn.example/result.png)'), path), ['https://cdn.example/result.png']);
    assert.deepEqual(imageRequestResults(chat([{ type: 'image_url', image_url: { url: 'data:image/png;base64,aGVsbG8=' } }]), path), ['data:image/png;base64,aGVsbG8=']);
    assert.deepEqual(imageRequestResults({ choices: [{ message: { images: [{ image_url: { url: 'https://cdn.example/result.png' } }] } }] }, path), ['https://cdn.example/result.png']);
    for (const content of ['我看到了一个穿白大褂的人', 'https://cdn.example/result.png', '![结果](javascript:alert(1))', '![原图](https://cdn.example/ref.png)']) {
        assert.throws(() => imageRequestResults(chat(content), path, imageReferenceValues.images), /未返回可识别的新图片/);
    }
});

test('multimodal image request performs one authenticated POST with exact references, without fallback on failure', async () => {
    const request = buildImageRequest(dualImageModel, imageReferenceValues, 'chat');
    const calls = [];
    const fetcher = async (url, init) => {
        calls.push({ url, body: JSON.parse(init.body) });
        assert.equal(init.headers.Authorization, 'Bearer mock-key');
        return new Response('{"error":{"message":"unsupported image generation"}}', { status: 400 });
    };
    await assert.rejects(api('mock-key', request.path, { body: request.body, fetcher }), /unsupported/);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, 'https://ghlink.top/v1/chat/completions');
    assert.equal(calls[0].body.messages[0].content[2].image_url.url, imageReferenceValues.images[0]);
});

for (const modelId of ['gpt-image-2.5-sunburst', 'gpt-image-2.5-flare', 'gpt-image-2']) for (const oldProtocol of ['', 'generations', 'chat']) {
test(`generator automatically submits ${modelId} once with references, ignoring old '${oldProtocol}' override`, async () => {
    // Exercise the actual event handlers and host operations without a browser or paid upstream.
    let cursor = 0;
    const hooks = [];
    const React = {
        createElement: (type, props, ...children) => ({ type, props: props || {}, children: children.flat(Infinity) }),
        useState: initial => { const index = cursor++; if (!(index in hooks)) hooks[index] = typeof initial === 'function' ? initial() : initial; return [hooks[index], value => { hooks[index] = typeof value === 'function' ? value(hooks[index]) : value; }]; },
        useRef: initial => { const index = cursor++; return hooks[index] ||= { current: initial }; },
        useEffect: () => {},
    };
    const plugin = createPlugin({ React });
    const nodes = [{ id: 'generator', type: 'ghlink-video:generator', width: 440, height: 760, position: { x: 0, y: 0 }, metadata: { ghMode: 'image', ghImageProtocol: oldProtocol, ghPrompt: '保持 @图片2 的人物衣服', ghMentionBindings: [{ token: '@图片2', nodeId: 'ref', identity: 'image:https://cdn.example/ref.png', kind: 'image', title: 'ref.png' }] } }];
    const ctx = {
        node: nodes[0], theme: { node: { stroke: '#ccc', text: '#222', muted: '#666' }, toolbar: { panel: '#fff' } },
        getNode: id => nodes.find(n => n.id === id), getNodes: () => nodes,
        getUpstream: () => [{ id: 'ref', type: 'image', title: 'ref.png', metadata: { content: 'https://cdn.example/ref.png' } }],
        applyOps: ops => { for (const op of ops) {
            if (op.type === 'add_node') nodes.push({ ...op, type: op.nodeType });
            if (op.type === 'update_node') { const node = nodes.find(n => n.id === op.id); node.metadata = { ...node.metadata, ...op.metadata }; }
        } },
    };
    const render = () => { cursor = 0; return plugin.nodes[0].Content({ ctx }); };
    const find = (tree, predicate) => {
        if (!tree || typeof tree !== 'object') return undefined;
        if (predicate(tree)) return tree;
        for (const child of tree.children || []) { const found = find(child, predicate); if (found) return found; }
    };
    const button = (tree, label) => find(tree, e => e.type === 'button' && e.children[0] === label);
    const calls = [];
    const realFetch = globalThis.fetch;
    globalThis.fetch = async (url, init) => {
        calls.push({ path: new URL(url).pathname, method: init.method, body: init.body ? JSON.parse(init.body) : null });
        if (url.endsWith('/models')) return new Response(JSON.stringify({ data: [{ id: modelId }] }));
        if (url.includes('/pricing?')) return new Response(JSON.stringify({ data: [{ model_name: modelId, supported_endpoint_types: dualImageModel.supported_endpoint_types, mapped_endpoints: [{ kind: 'image', path: '/v1/images/generations', parameters: dualImageModel.parameters }] }] }));
        assert.equal(new URL(url).pathname, '/v1/images/generations');
        return new Response(JSON.stringify({ data: [{ url: 'https://cdn.example/edited.png' }] }));
    };
    try {
        find(render(), e => e.type === 'input' && e.props.type === 'password').props.onChange({ target: { value: 'test-key-in-memory' } });
        await button(render(), '连接并刷新模型').props.onClick();
        assert.equal(find(render(), e => e.type === 'select' && e.children.some(c => c?.props?.value === 'chat')), undefined, 'no protocol selector in the UI');
        await button(render(), '生成图片').props.onClick();
        const posts = calls.filter(c => c.method === 'POST');
        assert.equal(posts.length, 1);
        assert.equal(nodes.length, 2);
        assert.equal(posts[0].body.model, modelId);
        assert.deepEqual(posts[0].body.images, ['https://cdn.example/ref.png']);
        assert.match(posts[0].body.prompt, /@图1/);
        assert.equal(posts[0].body.messages, undefined);
        assert.equal(nodes[1].metadata.ghResultURL, 'https://cdn.example/edited.png');
        assert.equal(nodes[1].metadata.ghProgress.phase, 'completed');
        assert.equal(nodes[0].metadata.ghLastRequest.images, 1);
        assert.ok(!JSON.stringify(nodes).includes('test-key-in-memory'));
    } finally { globalThis.fetch = realFetch; }
});
}
