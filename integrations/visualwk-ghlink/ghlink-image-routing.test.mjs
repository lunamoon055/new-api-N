import { test } from 'node:test';
import assert from 'node:assert/strict';
import createPlugin, { api, catalog, imageModelForReferences, buildImageRequest, prepareImageRequest, prepareReferences, compileMentions } from './ghlink-video.mjs';

const generation = { kind: 'image', type: 'image-generation', method: 'POST', path: '/v1/images/generations', parameters: { ratios: ['16:9', '1:1'], resolutions: ['1K', '4K'], ratio_field: 'ratio', resolution_field: 'resolution', max_images: 0, max_videos: 0, max_audios: 0 } };
const edit = { kind: 'image', type: 'image-edit', method: 'POST', path: '/v1/images/edits', parameters: { ...generation.parameters, image_field: 'image[]', min_images: 1, max_images: 20 } };
const imageModel = (endpoints = [generation, edit], id = 'gpt-image-2') => catalog({ data: [{ id }] }, { data: [{ model_name: id, mapped_endpoints: endpoints, supported_endpoint_types: ['image-generation'] }] }, 'image')[0];
const values = { prompt: '保留 @图1 的内容', ratio: '16:9', resolution: '4K', images: ['https://cdn.example/ref.png'] };
const signal = () => new AbortController().signal;
const png = () => new Blob(['reference-png-bytes'], { type: 'image/png' });

test('dual image endpoints are one authorized model regardless of endpoint order or exact alias', () => {
    for (const id of ['gpt-image-2', 'gpt-image-2-原生4K', 'future-dual-route-model']) for (const routes of [[generation, edit], [edit, generation]]) {
        const model = imageModel(routes, id);
        assert.equal(model.id, id);
        assert.equal(model.image_routes.length, 2);
        assert.equal(model.endpoint, generation.path);
        assert.equal(buildImageRequest(model, { prompt: '地球' }).path, generation.path);
        const request = buildImageRequest(model, values);
        assert.equal(request.path, edit.path);
        assert.deepEqual(request.body, { model: id, prompt: values.prompt, n: 1, ratio: '16:9', resolution: '4K', 'image[]': values.images });
    }
});

test('reference-capable generations retain their URL contract; route choice uses each endpoint limit', () => {
    const urlRoute = { ...generation, parameters: { ...generation.parameters, image_field: 'images', max_images: 1 } };
    const model = imageModel([edit, urlRoute]);
    assert.equal(buildImageRequest(model, values).path, generation.path);
    assert.deepEqual(buildImageRequest(model, values).body.images, values.images);
    const two = { ...values, images: [...values.images, 'https://cdn.example/second.png'] };
    assert.equal(buildImageRequest(model, two).path, edit.path);
    assert.equal(imageModelForReferences(model, 2).parameters.max_images, 20);
    const singular = imageModel([{ ...urlRoute, parameters: { image_field: 'image_url' } }, edit]);
    assert.equal(buildImageRequest(singular, two).path, edit.path);
    assert.throws(() => buildImageRequest(model, { ...values, images: Array.from({ length: 21 }, (_, i) => `https://cdn.example/${i}.png`) }), /最多支持 20/);
    assert.throws(() => buildImageRequest(imageModel([generation]), values), /最多支持 0/);
});

test('edit-only models are recognized, require references and never guess custom routes or file fields', () => {
    assert.equal(buildImageRequest(imageModel([edit]), values).path, edit.path);
    assert.throws(() => buildImageRequest(imageModel([edit]), { prompt: '地球' }), /至少需要 1/);
    for (const endpoint of [{ ...edit, path: '/vendor/images/edit' }, { ...edit, method: 'GET' }, { ...edit, parameters: { image_field: 'guess' } }]) {
        assert.equal(imageModel([endpoint]), undefined);
    }
    const p = { ...generation.parameters, image_field: 'images', unknown_fields: ['max_images'] };
    assert.equal(buildImageRequest(imageModel([{ ...generation, parameters: p }, edit]), values).path, generation.path);
});

test('multipart sends actual reference bytes and endpoint-specific fields with auth only on GH POST', async () => {
    const selected = imageModelForReferences(imageModel(), 1);
    const downloads = [];
    const request = await prepareImageRequest(buildImageRequest(selected, values), selected.parameters, signal(), { fetcher: async (url, options) => {
        downloads.push({ url, options }); return new Response(png());
    } });
    assert.equal(downloads.length, 1);
    assert.equal(downloads[0].options.credentials, 'omit');
    assert.equal(downloads[0].options.redirect, 'error');
    assert.equal(downloads[0].options.headers, undefined);
    assert.ok(request.body instanceof FormData);
    assert.equal(request.body.get('resolution'), '4K');
    assert.equal(request.body.get('ratio'), '16:9');
    assert.equal(request.body.get('images'), null);
    assert.equal(await request.body.get('image[]').text(), await png().text());
    const calls = [];
    await assert.rejects(api('fixture-private-key', request.path, { body: request.body, fetcher: async (url, options) => {
        calls.push({ url, options }); return new Response('{"error":{"message":"edit failed"}}', { status: 400 });
    } }), /edit failed/);
    assert.equal(calls.length, 1);
    assert.equal(calls[0].url, 'https://ghlink.top/v1/images/edits');
    assert.equal(calls[0].options.headers.Authorization, 'Bearer fixture-private-key');
    assert.equal(calls[0].options.headers['Content-Type'], undefined, 'browser creates the multipart boundary');
});

test('local edit references bypass public upload, retain deduplication and stable @ bindings', async () => {
    const item = { id: 'ref', kind: 'image', source: 'blob:expired', storageKey: 'image:fixture', identity: 'image:image:fixture', title: '人物.png', mimeType: 'image/png' };
    const selected = imageModelForReferences(imageModel(), 1), files = new Map(), resolved = new Map();
    const refs = await prepareReferences([item, { ...item, id: 'duplicate' }], {}, signal(), { files, parameters: selected.parameters, read: async () => png(), upload: async () => assert.fail('edit files must not upload separately'), onResolved: (input, url) => resolved.set(input.id, url) });
    assert.equal(refs.images.length, 1);
    const prompt = compileMentions('保留 @图片2', [{ token: '@图片2', nodeId: 'ref', identity: item.identity, kind: 'image' }], [item], refs, resolved);
    const prepared = await prepareImageRequest(buildImageRequest(selected, { ...values, ...refs, prompt }), selected.parameters, signal(), { files, fetcher: async () => assert.fail('local files must not download') });
    assert.equal(prepared.body.getAll('image[]').length, 1);
    assert.equal(prepared.body.get('prompt'), '保留 @图1');
    assert.equal(await prepared.body.get('image[]').text(), await png().text());
    assert.ok(![...prepared.body.entries()].some(([, value]) => typeof value === 'string' && value.includes('local.invalid')));
});

test('bad, unreadable, oversized or cancelled edit files fail before a generation POST', async () => {
    const selected = imageModelForReferences(imageModel(), 1), request = buildImageRequest(selected, values);
    for (const response of [new Response('missing', { status: 404 }), new Response('<html>error</html>', { headers: { 'Content-Type': 'text/html' } }), new Response(new Blob([], { type: 'image/png' })), new Response(new Blob([new Uint8Array(1024 * 1024 + 1)], { type: 'image/png' }))]) {
        await assert.rejects(prepareImageRequest(request, { ...selected.parameters, max_image_size_mb: 1 }, signal(), { fetcher: async () => response }), /无法读取|须为|为空|超过/);
    }
    await assert.rejects(prepareImageRequest(request, selected.parameters, signal(), { fetcher: async () => { throw new TypeError('CORS blocked'); } }), /CORS/);
    const controller = new AbortController();
    await assert.rejects(prepareImageRequest(request, selected.parameters, controller.signal, { fetcher: async (_, options) => {
        controller.abort(); assert.equal(options.signal.aborted, true); return new Response(png());
    } }), { name: 'AbortError' });
    const plain = buildImageRequest(imageModel(), { prompt: '地球' });
    assert.equal(await prepareImageRequest(plain, generation.parameters, signal(), { fetcher: () => assert.fail('no-reference generation must not fetch files') }), plain);
});

// Execute actual Content handlers, not only the route/request helpers.
function app(references, manual = '', prompt = '保留 @图片1') {
    let cursor = 0;
    const hooks = [], React = {
        createElement: (type, props, ...children) => ({ type, props: props || {}, children: children.flat(Infinity) }),
        useState(initial) { const index = cursor++; if (!(index in hooks)) hooks[index] = typeof initial === 'function' ? initial() : initial; return [hooks[index], value => { hooks[index] = typeof value === 'function' ? value(hooks[index]) : value; }]; },
        useRef(initial) { const index = cursor++; return hooks[index] ||= { current: initial }; }, useEffect() {},
    };
    const plugin = createPlugin({ React }), nodes = [{ id: 'generator', type: 'ghlink-video:generator', position: { x: 0, y: 0 }, width: 440, height: 760, metadata: { ghMode: 'image', ghPrompt: prompt, ghImages: manual, ghImageProtocol: 'chat' } }];
    const ctx = { get node() { return nodes[0]; }, theme: { node: {}, toolbar: {} }, getNode: id => nodes.find(n => n.id === id), getNodes: () => nodes, getUpstream: () => references,
        applyOps(ops) { for (const op of ops) { if (op.type === 'add_node') nodes.push({ ...op, type: op.nodeType }); if (op.type === 'update_node') { const n = nodes.find(n => n.id === op.id); n.metadata = { ...n.metadata, ...op.metadata }; } } },
    };
    const render = () => { cursor = 0; return plugin.nodes[0].Content({ ctx }); };
    function find(tree, predicate) { if (!tree || typeof tree !== 'object') return; if (predicate(tree)) return tree; for (const child of tree.children || []) { const match = find(child, predicate); if (match) return match; } }
    const button = label => find(render(), e => e.type === 'button' && e.children[0] === label);
    return { nodes, render, button, find, key: () => find(render(), e => e.type === 'input' && e.props.type === 'password').props.onChange({ target: { value: 'fixture-private-key' } }) };
}
const publicReference = { id: 'ref', type: 'image', title: 'ref.png', metadata: { content: values.images[0] } };

for (const scenario of ['connected', 'manual', 'none', 'media-failure', 'post-failure', 'over-limit']) test(`real generator uses automatic image routing: ${scenario}`, async () => {
    const manual = scenario === 'manual' ? values.images[0] : scenario === 'over-limit' ? Array.from({ length: 21 }, (_, i) => `https://cdn.example/${i}.png`).join('\n') : '';
    const view = app(['connected', 'media-failure', 'post-failure'].includes(scenario) ? [publicReference] : [], manual, scenario === 'none' || scenario === 'over-limit' ? '地球' : '保留 @图片1');
    const original = globalThis.fetch, posts = [], gets = [];
    globalThis.fetch = async (url, init) => {
        if (url.endsWith('/models')) return new Response(JSON.stringify({ data: [{ id: 'gpt-image-2' }] }));
        if (url.includes('/pricing?')) return new Response(JSON.stringify({ data: [{ model_name: 'gpt-image-2', mapped_endpoints: [generation, edit] }] }));
        if (init.method !== 'POST') { gets.push(url); return scenario === 'media-failure' ? new Response('missing', { status: 404 }) : new Response(png()); }
        posts.push({ url, init });
        return scenario === 'post-failure' ? new Response('{"error":{"message":"read model specification failed"}}', { status: 400 }) : new Response('{"data":[{"url":"https://cdn.example/result.png"}]}');
    };
    try {
        view.key(); await view.button('连接并刷新模型').props.onClick();
        const imageInput = view.find(view.render(), e => e.type === 'label' && String(e.children[0]?.children[0]).startsWith('图片 · 最多 20'));
        assert.equal(imageInput.children[1].props.disabled, false, 'manual references remain available before connecting anything');
        assert.equal(view.find(view.render(), e => e.type === 'select' && e.children.some(c => c?.props?.value === 'chat')), undefined);
        await view.button('生成图片').props.onClick();
        if (['over-limit', 'media-failure'].includes(scenario)) {
            assert.equal(posts.length, 0); assert.equal(gets.length, scenario === 'over-limit' ? 0 : 1);
            assert.equal(view.nodes[0].metadata.ghPhase, undefined); assert.equal(view.nodes[1].metadata.ghProgress.phase, 'error');
        } else {
            assert.equal(posts.length, 1);
            const plain = scenario === 'none';
            assert.equal(posts[0].url, 'https://ghlink.top' + (plain ? generation.path : edit.path));
            if (plain) assert.equal(JSON.parse(posts[0].init.body).prompt, '地球');
            else { assert.ok(posts[0].init.body instanceof FormData); assert.equal(posts[0].init.body.getAll('image[]').length, 1); assert.equal(posts[0].init.body.get('prompt'), '保留 @图1'); }
            assert.equal(view.nodes[1].metadata.ghProgress.phase, scenario === 'post-failure' ? 'uncertain' : 'completed');
            if (scenario === 'post-failure') assert.equal(view.nodes[0].metadata.ghPhase, 'uncertain');
        }
        assert.ok(!JSON.stringify(view.nodes).includes('fixture-private-key'));
    } finally { globalThis.fetch = original; }
});
