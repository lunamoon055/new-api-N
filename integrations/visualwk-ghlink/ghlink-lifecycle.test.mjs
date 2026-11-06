import { test } from 'node:test';
import assert from 'node:assert/strict';
import createPlugin from './ghlink-video.mjs';

// Run the real node handlers with effect cleanup and fresh hook state on every
// remount. Network responses are held until after the view has been replaced.
function harness(kind = 'image') {
    let mounted;
    const React = {
        createElement: (type, props, ...children) => ({ type, props: props || {}, children: children.flat(Infinity) }),
        useState(initial) {
            const view = mounted, index = view.cursor++;
            if (!(index in view.hooks)) view.hooks[index] = typeof initial === 'function' ? initial() : initial;
            return [view.hooks[index], value => { if (!view.disposed) view.hooks[index] = typeof value === 'function' ? value(view.hooks[index]) : value; }];
        },
        useRef(initial) { const index = mounted.cursor++; return mounted.hooks[index] ||= { current: initial }; },
        useEffect(setup, deps) {
            const view = mounted, index = view.cursor++, previous = view.effects[index];
            if (!previous || deps.some((value, i) => value !== previous.deps[i])) view.pending.push(() => {
                previous?.cleanup?.(); view.effects[index] = { setup, deps, cleanup: setup() };
            });
        },
    };
    const plugin = createPlugin({ React });
    const dispose = plugin.setup();
    const blobs = new Map();
    const nodes = new Map([['source', { id: 'source', type: 'ghlink-video:generator', width: 440, height: 760, position: { x: 0, y: 0 }, metadata: { ghMode: kind, ghPrompt: 'test', ghSeconds: '5' } }]]);
    const find = (tree, predicate) => {
        if (!tree || typeof tree !== 'object') return;
        if (predicate(tree)) return tree;
        for (const child of tree.children || []) { const found = find(child, predicate); if (found) return found; }
    };
    let onOps = () => {};
    const context = (id = 'source') => ({
        node: nodes.get(id), theme: { node: { stroke: '#ccc', text: '#222', muted: '#666' }, toolbar: { panel: '#fff' } },
        getNode: id => nodes.get(id), getNodes: () => [...nodes.values()], getUpstream: () => [],
        storage: { get: async key => blobs.get(key), set: async (key, value) => { blobs.set(key, value); } },
        applyOps: ops => {
            for (const op of ops) {
                if (op.type === 'add_node') nodes.set(op.id, { ...op, type: op.nodeType });
                if (op.type === 'update_node' && nodes.has(op.id)) {
                    const node = nodes.get(op.id); nodes.set(op.id, { ...node, metadata: { ...node.metadata, ...op.metadata } });
                }
            }
            onOps(ops);
        },
    });
    let view;
    function render() {
        mounted = view; view.cursor = 0;
        const tree = plugin.nodes[0].Content({ ctx: context() });
        for (const effect of view.pending.splice(0)) effect();
        return tree;
    }
    function unmount() { if (view) { view.disposed = true; for (const effect of view.effects) effect?.cleanup?.(); } }
    function mount() { unmount(); view = { hooks: [], effects: [], pending: [], cursor: 0, disposed: false }; return render(); }
    const button = label => find(render(), element => element.type === 'button' && element.children[0] === label);
    const recoveryInput = id => {
        mounted = { hooks: [], effects: [], pending: [], cursor: 0 };
        return find(plugin.nodes[1].Content({ ctx: context(id) }), element => element.type === 'input' && element.props.type === 'file');
    };
    function key(value) { find(render(), element => element.type === 'input' && element.props.type === 'password').props.onChange({ target: { value } }); }
    mount();
    return { nodes, blobs, recoveryInput, render, mount, unmount, key, button, dispose: () => { unmount(); dispose(); },
        onOps: callback => { onOps = callback; },
        replayEffects: () => { for (const effect of view.effects) if (effect) { effect.cleanup?.(); effect.cleanup = effect.setup(); } },
        result: () => [...nodes.values()].find(node => node.type === 'ghlink-video:result'),
    };
}

const json = value => new Response(JSON.stringify(value));
async function until(predicate) {
    for (let index = 0; index < 100; index++) { if (predicate()) return; await new Promise(resolve => setImmediate(resolve)); }
    assert.fail('expected request state was not reached');
}
function mockNetwork(kind = 'image') {
    const original = globalThis.fetch, posts = [], polls = [], pending = [];
    globalThis.fetch = async (url, init) => {
        if (url.endsWith('/models')) return json({ data: [{ id: 'fixture-model' }] });
        if (url.includes('/pricing?')) return json({ data: [{ model_name: 'fixture-model', mapped_endpoints: [{ kind, path: kind === 'image' ? '/v1/images/generations' : '/v1/videos', parameters: { durations: [5] } }] }] });
        const call = { url, signal: init.signal };
        (init.method === 'POST' ? posts : polls).push(call);
        return new Promise((resolve, reject) => {
            const abort = () => reject(init.signal.reason);
            call.reply = value => { init.signal.removeEventListener('abort', abort); resolve(json(value)); };
            call.disconnect = () => { init.signal.removeEventListener('abort', abort); reject(new TypeError('Failed to fetch')); };
            pending.push(call);
            if (init.signal.aborted) abort(); else init.signal.addEventListener('abort', abort, { once: true });
        });
    };
    return { posts, polls, restore: () => { for (const call of pending) call.reply({}); globalThis.fetch = original; } };
}
async function connect(app) { app.key('fixture-private-key'); await app.button('连接并刷新模型').props.onClick(); }

test('creating a result can remount the generator before POST without interrupting submission', async () => {
    const network = mockNetwork(), app = harness();
    let task;
    try {
        await connect(app);
        app.onOps(ops => { if (ops.some(op => op.type === 'add_node')) queueMicrotask(() => app.mount()); });
        task = app.button('生成图片').props.onClick();
        await until(() => network.posts.length === 1);
        assert.equal(network.posts[0].signal.aborted, false);
        network.posts[0].reply({ data: [{ url: 'https://cdn.example/image.png' }] });
        await task;
        assert.equal(app.result().metadata.ghProgress.phase, 'completed');
        assert.equal(network.posts.length, 1);
    } finally { app.dispose(); network.restore(); await task; }
});

test('image request, model catalog and duplicate-submit lock survive repeated remounts and effect cleanup', async () => {
    const network = mockNetwork(), app = harness();
    let task;
    try {
        await connect(app);
        task = app.button('生成图片').props.onClick();
        await until(() => network.posts.length === 1);
        const resultId = app.result().id;
        for (let index = 0; index < 3; index++) { app.mount(); app.replayEffects(); }
        assert.equal(network.posts[0].signal.aborted, false);
        assert.equal(app.button('生成图片').props.disabled, true);
        assert.ok(app.button('停止本页等待'));
        // An already queued click must also be rejected by the shared lock.
        await app.button('生成图片').props.onClick();
        network.posts[0].reply({ data: [{ url: 'https://cdn.example/image.png' }] });
        await task;
        assert.equal(network.posts.length, 1);
        assert.equal(app.result().id, resultId);
        assert.equal(app.result().metadata.ghProgress.phase, 'completed');
        assert.equal(app.result().metadata.ghResultURL, 'https://cdn.example/image.png');
        assert.ok(JSON.stringify(app.render()).includes('fixture-model'));
        assert.ok(!JSON.stringify([...app.nodes.values()]).includes('fixture-private-key'));
    } finally { app.dispose(); network.restore(); await task; }
});

test('a stale generation callback cannot send a second POST after the first one completes', async () => {
    const network = mockNetwork(), app = harness();
    try {
        await connect(app);
        const click = app.button('生成图片').props.onClick;
        const task = click();
        await until(() => network.posts.length === 1);
        network.posts[0].reply({ data: [{ url: 'https://cdn.example/image.png' }] });
        await task;
        await click();
        assert.equal(network.posts.length, 1);
        assert.equal(app.nodes.size, 2);
    } finally { app.dispose(); network.restore(); }
});

test('a dropped sync image response stays locked and a recovered file fills the existing result without a POST', async () => {
    const network = mockNetwork(), app = harness();
    const originalDecoder = globalThis.createImageBitmap;
    globalThis.createImageBitmap = async () => ({ close() {} });
    try {
        await connect(app);
        const click = app.button('生成图片').props.onClick;
        const task = click();
        await until(() => network.posts.length === 1);
        network.posts[0].disconnect();
        await task;
        app.mount();
        assert.equal(app.nodes.get('source').metadata.ghPhase, 'uncertain');
        assert.equal(app.result().metadata.ghProgress.phase, 'uncertain');
        assert.equal(app.button('生成图片').props.disabled, true);
        assert.equal(app.button('新任务'), undefined);
        await click();
        const id = app.result().id;
        const file = new Blob(['fixture-image'], { type: 'image/png' });
        await app.recoveryInput(id).props.onChange({ currentTarget: { files: [file], value: 'image.png' } });
        assert.equal(app.nodes.get('source').metadata.ghPhase, 'completed');
        assert.equal(app.nodes.get(id).metadata.ghProgress.phase, 'completed');
        assert.equal(app.blobs.get(app.nodes.get(id).metadata.ghBlobKey), file);
        assert.equal(app.nodes.size, 2);
        assert.equal(network.posts.length, 1);
    } finally { app.dispose(); network.restore(); globalThis.createImageBitmap = originalDecoder; }
});

test('restoring an older failed image does not change a newer task or unlock it', async () => {
    const app = harness();
    const originalDecoder = globalThis.createImageBitmap;
    globalThis.createImageBitmap = async () => ({ close() {} });
    try {
        const source = app.nodes.get('source');
        Object.assign(source.metadata, { ghPhase: 'uncertain', ghResultNodeId: 'newer' });
        app.nodes.set('older', { id: 'older', metadata: { ghMode: 'image', ghSourceNodeId: 'source', ghLastRequest: { path: '/v1/images/edits' }, ghProgress: { phase: 'error' } } });
        const input = app.recoveryInput('older');
        const bad = new Blob(['not an image'], { type: 'text/plain' });
        await input.props.onChange({ currentTarget: { files: [bad], value: 'bad.txt' } });
        assert.equal(app.blobs.size, 0);
        const file = new Blob(['fixture-image'], { type: 'image/png' });
        await input.props.onChange({ currentTarget: { files: [file], value: 'image.png' } });
        assert.equal(app.nodes.get('older').metadata.ghProgress.phase, 'completed');
        assert.equal(app.nodes.get('source').metadata.ghPhase, 'uncertain');
        assert.equal(app.nodes.get('source').metadata.ghResultNodeId, 'newer');
    } finally { app.dispose(); globalThis.createImageBitmap = originalDecoder; }
});

test('video POST response and polling complete in the same result after separate remounts', async () => {
    const network = mockNetwork('video'), app = harness('video');
    let task;
    try {
        await connect(app);
        task = app.button('生成视频').props.onClick();
        await until(() => network.posts.length === 1);
        app.mount();
        assert.equal(network.posts[0].signal.aborted, false);
        network.posts[0].reply({ id: 'video-accepted', status: 'queued' });
        await until(() => network.polls.length === 1);
        assert.equal(app.nodes.get('source').metadata.ghTaskId, 'video-accepted');
        app.mount();
        assert.equal(network.polls[0].signal.aborted, false);
        network.polls[0].reply({ status: 'completed', url: 'https://cdn.example/result.mp4' });
        await task;
        assert.equal(app.result().metadata.ghProgress.phase, 'completed');
        assert.equal(app.result().metadata.ghResultURL, 'https://cdn.example/result.mp4');
        assert.equal(network.posts.length, 1);
    } finally { app.dispose(); network.restore(); await task; }
});

test('stop after remount aborts the original image request, preserves uncertainty and never resubmits', async () => {
    const network = mockNetwork(), app = harness();
    let task;
    try {
        await connect(app);
        task = app.button('生成图片').props.onClick();
        await until(() => network.posts.length === 1);
        app.mount();
        assert.equal(network.posts[0].signal.aborted, false);
        app.button('停止本页等待').props.onClick();
        await task;
        assert.equal(network.posts[0].signal.aborted, true);
        assert.equal(app.nodes.get('source').metadata.ghPhase, 'uncertain');
        assert.equal(app.result().metadata.ghProgress.phase, 'paused');
        app.mount();
        assert.equal(app.button('生成图片').props.disabled, true);
        assert.equal(network.posts.length, 1);
    } finally { app.dispose(); network.restore(); await task; }
});

test('source deletion or plugin disposal stops detached work and does not write a fake completion', async () => {
    for (const remove of [false, true]) {
        const network = mockNetwork(), app = harness();
        let task;
        try {
            await connect(app);
            task = app.button('生成图片').props.onClick();
            await until(() => network.posts.length === 1);
            app.unmount();
            if (remove) app.nodes.delete('source'); else app.dispose();
            await task;
            assert.equal(network.posts[0].signal.aborted, true);
            assert.notEqual(app.result().metadata.ghProgress.phase, 'completed');
            assert.equal(network.posts.length, 1);
        } finally { app.dispose(); network.restore(); await task; }
    }
});

test('reload warning exists only while a generation is active and is removed when it finishes', async () => {
    const originalWindow = globalThis.window, handlers = new Map();
    globalThis.window = { addEventListener: (type, handler) => handlers.set(type, handler), removeEventListener: type => handlers.delete(type) };
    const network = mockNetwork(), app = harness();
    let task;
    try {
        await connect(app);
        assert.equal(handlers.has('beforeunload'), false);
        task = app.button('生成图片').props.onClick();
        await until(() => network.posts.length === 1);
        app.mount();
        let prevented = false;
        const event = { preventDefault: () => { prevented = true; } };
        handlers.get('beforeunload')(event);
        assert.equal(prevented, true);
        assert.equal(event.returnValue, '');
        network.posts[0].reply({ data: [{ url: 'https://cdn.example/image.png' }] });
        await task;
        assert.equal(handlers.has('beforeunload'), false);
    } finally { app.dispose(); network.restore(); await task; globalThis.window = originalWindow; }
});
