// GHLINK video node for Infinite Canvas / 视觉工坊. No bundled credentials or dependencies.
const ORIGIN = 'https://ghlink.top';
const DONE = new Set(['completed', 'succeeded', 'success', 'settled', 'done']);
const FAILED = new Set(['failed', 'failure', 'error', 'cancelled', 'canceled', 'released']);
const unwrap = value => value?.data && !Array.isArray(value.data) ? value.data : value;
const strings = value => Array.isArray(value) ? value.filter(x => typeof x === 'string' && x.trim()) : [];
const statusOf = value => String(value?.status || value?.state || '').toLowerCase();
const safeURL = value => {
    try { const url = new URL(value); return ['http:', 'https:'].includes(url.protocol) && !url.username && !url.password ? url.href : ''; }
    catch { return ''; }
};
function publicError(error, key = '') {
    let message = String(error?.message || error || '请求失败');
    if (key) message = message.split(key).join('[已隐藏]');
    return message.replace(/sk-[\w-]+/g, '[已隐藏]').slice(0, 600);
}

export async function api(key, path, { body, signal, binary = false, fetcher = fetch, timeoutMs } = {}) {
    if (!key?.trim()) throw new Error('请先填写 GH API Key');
    if (!path.startsWith('/v1/')) throw new Error('不支持的接口');
    const form = typeof FormData !== 'undefined' && body instanceof FormData;
    const response = await fetcher(ORIGIN + path, {
        method: body === undefined ? 'GET' : 'POST', credentials: 'omit', redirect: 'error', cache: 'no-store',
        headers: { Authorization: `Bearer ${key.trim()}`, ...(body !== undefined && !form ? { 'Content-Type': 'application/json' } : {}) },
        body: body === undefined ? undefined : form ? body : JSON.stringify(body),
        signal: signal ? AbortSignal.any([signal, AbortSignal.timeout(timeoutMs || (body === undefined ? 60000 : 120000))]) : AbortSignal.timeout(timeoutMs || 60000),
    });
    if (binary && response.ok) {
        const blob = await response.blob();
        if (!blob.size || /json|text\/html/.test(blob.type)) throw new Error('内容接口未返回视频文件');
        return blob;
    }
    const payload = await response.json().catch(() => null);
    if (!response.ok || !payload || payload.success === false || payload.error) {
        const error = new Error(publicError(payload?.error?.message || payload?.message || `请求失败（HTTP ${response.status}）`, key));
        error.status = response.status;
        const retry = response.headers.get('Retry-After');
        error.retryMs = retry ? (Number.isFinite(Number(retry)) ? Number(retry) * 1000 : Math.max(0, Date.parse(retry) - Date.now())) : 0;
        throw error;
    }
    return payload;
}

export function catalog(models, pricing, kind = 'video') {
    if (!Array.isArray(models?.data) || !Array.isArray(pricing?.data)) throw new Error('模型或价格目录格式不正确');
    const authorized = new Set(models.data.map(item => item.id));
    return pricing.data.filter(item => authorized.has(item.model_name)).flatMap(item => {
        const path = kind === 'image' ? '/v1/images/generations' : '/v1/videos';
        const endpoint = item.mapped_endpoints?.find(e => e.kind === kind && e.path === path);
        if (!endpoint && !item.supported_endpoint_types?.includes(kind === 'image' ? 'image-generation' : 'openai-video')) return [];
        if (!endpoint && item.mapped_endpoints?.some(e => e.kind === kind)) return [];
        return [{ ...item, id: item.model_name, kind, endpoint: path, parameters: endpoint?.parameters || {}, currency: pricing.currency || 'CNY', price_scope: pricing.price_scope }];
    });
}

const FIELD_NAMES = {
    duration: ['seconds', 'duration'], ratio: ['ratio', 'aspect_ratio'], resolution: ['resolution', 'size', 'output_resolution'],
    image: ['images', 'referenceImages', 'reference_images', 'image_urls', 'reference_image_urls', 'image_url'],
    video: ['videos', 'referenceVideos', 'reference_videos', 'video_urls', 'reference_video_urls', 'video_url'],
    audio: ['audios', 'referenceAudios', 'reference_audios', 'audio_urls', 'audio_reference', 'audio_url'],
};
function field(parameters, kind, fallback) {
    const value = parameters[`${kind}_field`] || fallback;
    if (!FIELD_NAMES[kind].includes(value)) throw new Error(`该模型的 ${kind} 字段暂不支持：${value}`);
    return value;
}
export function buildPayload(model, values) {
    if (!model) throw new Error('请先刷新并选择模型');
    if (model.price_configured === false) throw new Error('模型尚未配置价格');
    const p = model.parameters;
    const prompt = String(values.prompt || '').trim();
    const image = model.kind === 'image';
    const seconds = Number(values.seconds);
    if (!prompt) throw new Error('请输入提示词');
    if (!image && (!Number.isSafeInteger(seconds) || seconds < 1)) throw new Error('请填写正整数时长');
    if (!image && p.durations?.length && !p.durations.includes(seconds)) throw new Error('时长不在模型支持范围内');
    const payload = { model: model.id, prompt, ...(image ? { n: 1 } : { [field(p, 'duration', 'seconds')]: seconds }) };
    for (const [kind, options] of [['ratio', p.ratios], ['resolution', p.resolutions]]) {
        if (kind === 'ratio' && (p.omit_ratio || (image && !p.ratio_field))) continue;
        const value = kind === 'resolution' && p.request_resolution ? p.request_resolution : values[kind];
        if (!value) continue;
        if (!(kind === 'resolution' && p.request_resolution) && options?.length && !options.includes(value)) throw new Error(`${kind} 不在模型支持范围内`);
        payload[field(p, kind, kind === 'resolution' && image ? 'size' : kind)] = value;
    }
    const counts = {};
    for (const [kind, plural] of [['image', 'images'], ['video', 'videos'], ['audio', 'audios']]) {
        const urls = referenceURLs(values[plural]);
        if (urls.some(url => !safeURL(url))) throw new Error('参考素材必须是 HTTP(S) 链接，每行一个');
        counts[plural] = urls.length;
        const maximum = p[`max_${plural}`];
        if (!p.capabilities_unknown && typeof maximum === 'number' && urls.length > maximum) throw new Error(`${plural} 最多支持 ${maximum} 个参考素材（已连接和手填素材合计 ${urls.length} 个）`);
        if (image && urls.length && (kind !== 'image' || !p.image_field)) throw new Error('该图片模型未声明此类参考素材接口，请选择支持参考图的模型；素材不会被忽略');
        if (urls.length) {
            const name = field(p, kind, plural);
            if (name.endsWith('_url') && urls.length > 1) throw new Error(`${kind} 字段只接受一个素材`);
            payload[name] = name.endsWith('_url') ? urls[0] : urls;
        }
    }
    if (counts.images < (p.min_images || 0)) throw new Error(`至少需要 ${p.min_images} 张参考图`);
    if ((p.audio_needs_image || p.audio_requires_image) && counts.audios && !counts.images) throw new Error('该模型使用参考音频时必须提供参考图');
    return payload;
}

export function supportsImageChat(model) {
    return model?.kind === 'image' && (model.supported_endpoint_types?.includes('openai') || model.mapped_endpoints?.some(x => x.path === '/v1/chat/completions'));
}

export function buildImageRequest(model, values, protocol = '') {
    if (!model || model.kind !== 'image') throw new Error('请先选择图片模型');
    if (protocol && !['generations', 'chat'].includes(protocol)) throw new Error('不支持的图片调用方式');
    const images = referenceURLs(values.images);
    // Reuse bounds checks, but do not require a generations-only image field for chat.
    const validated = buildPayload(protocol === 'chat' ? { ...model, parameters: { ...model.parameters, image_field: 'images' } } : model, values);
    if (protocol !== 'chat') return { path: '/v1/images/generations', body: validated, referenceCount: images.length };
    if (!supportsImageChat(model)) throw new Error('模型目录未声明对话接口，无法使用对话参考图方式');
    const instructions = ['请生成图片并返回图片结果。', values.ratio ? `画面比例：${values.ratio}。` : '', values.resolution ? `期望分辨率：${values.resolution}。` : '', validated.prompt].filter(Boolean).join('\n');
    const content = [{ type: 'text', text: instructions }];
    images.forEach((url, index) => content.push({ type: 'text', text: `参考图片 @图${index + 1}：` }, { type: 'image_url', image_url: { url } }));
    return { path: '/v1/chat/completions', body: { model: model.id, messages: [{ role: 'user', content }], stream: false }, referenceCount: images.length };
}

export function imageRequestResults(response, path, references = []) {
    if (path !== '/v1/chat/completions') return imageResults(response);
    const outputs = [];
    const add = raw => {
        const value = typeof raw === 'string' ? raw : raw?.url;
        const url = safeURL(value) || (/^data:image\/(png|jpeg|webp);base64,[A-Za-z0-9+/=\s]+$/i.test(value || '') ? value : '');
        if (url && !references.includes(url)) outputs.push(url);
    };
    const textImages = text => { for (const match of String(text || '').matchAll(/!\[[^\]]*\]\(<?(https?:\/\/[^\s<>]+?|data:image\/(?:png|jpeg|webp);base64,[A-Za-z0-9+/=]+)>?(?:\s+"[^"\n]*")?\)/gi)) add(match[1]); };
    for (const choice of response?.choices || []) {
        const message = choice.message || {};
        if (typeof message.content === 'string') textImages(message.content);
        else if (Array.isArray(message.content)) for (const part of message.content) {
            if (part.type === 'image_url') add(part.image_url);
            else if (part.type === 'image') add(part.url || part.image_url);
            else if (part.type === 'text') textImages(part.text);
        }
        if (Array.isArray(message.images)) for (const part of message.images) add(part.image_url || part.url);
    }
    if (!outputs.length) throw new Error('对话接口未返回可识别的新图片（可能只返回文字或原参考图）。没有自动重试，请核对 GH 记录；接口支持对话不代表已验证参考图生成');
    return [...new Set(outputs)];
}

function referenceURLs(value) {
    return [...new Set((Array.isArray(value) ? value : String(value || '').split('\n')).map(x => x.trim()).filter(Boolean))];
}

const MENTION_NAMES = { image: '图片', video: '视频', audio: '音频' };
const mentionPattern = () => /@(图片|图|视频|音频)([1-9]\d*)(?!\d)/g;
export function mentionAt(value, cursor) {
    const prefix = value.slice(0, cursor);
    const match = /[@＠]([^\s@＠，。；：、()[\]{}]{0,80})$/.exec(prefix);
    if (!match || /[a-zA-Z0-9_.+-]/.test(prefix[match.index - 1] || '')) return null;
    return { start: match.index, end: cursor, query: match[1] };
}

// Labels remain bound to a node, rather than silently shifting when connections change.
export function mentionOptions(items, bindings = []) {
    const reserved = new Set(bindings.map(x => x.token));
    return items.filter(x => MENTION_NAMES[x.kind]).map(item => {
        const bound = bindings.find(x => x.kind === item.kind && (x.nodeId === item.id || x.identity === item.identity));
        let index = 1;
        while (reserved.has(`@${MENTION_NAMES[item.kind]}${index}`)) index++;
        const token = bound?.token || `@${MENTION_NAMES[item.kind]}${index}`;
        reserved.add(token);
        return { ...item, token };
    });
}

export function insertMention(value, range, item, bindings = []) {
    const insertion = `${item.token} `;
    return {
        prompt: value.slice(0, range.start) + insertion + value.slice(range.end),
        cursor: range.start + insertion.length,
        bindings: [...bindings.filter(x => x.token !== item.token), { token: item.token, nodeId: item.id, identity: item.identity, kind: item.kind, title: item.title }],
    };
}

export function compileMentions(prompt, bindings, items, references, resolvedURLs) {
    const legends = new Set();
    const output = String(prompt).replace(mentionPattern(), (token, label, number) => {
        const kind = label === '图' || label === '图片' ? 'image' : label === '视频' ? 'video' : 'audio';
        const bound = bindings.find(x => x.token === token);
        let index = Number(number);
        if (bound) {
            const item = items.find(x => x.kind === bound.kind && (x.id === bound.nodeId || x.identity === bound.identity));
            if (!item) throw new Error(`引用素材「${bound.title}」已断开或不存在，请重新连接，或删除提示词中的 ${token}`);
            index = references[`${kind}s`].indexOf(resolvedURLs.get(item.id)) + 1;
        }
        if (index < 1 || index > references[`${kind}s`].length) throw new Error(`${token} 没有对应的参考素材，请输入 @ 重新选择`);
        const canonical = `@${kind === 'image' ? '图' : MENTION_NAMES[kind]}${index}`;
        if (kind !== 'image') legends.add(`${canonical} 对应上传的第 ${index} 个参考${MENTION_NAMES[kind]}`);
        return canonical;
    });
    return output + (legends.size ? `\n素材引用：${[...legends].join('；')}。` : '');
}

// Only direct incoming resources are read. Do not scan unrelated canvas nodes or reverse links.
export function connectedResources(nodes) {
    const seen = new Set();
    return nodes.flatMap(node => {
        const m = node.metadata || {};
        const kind = ['image', 'video', 'audio', 'text'].includes(node.type) ? node.type : ['ghlink-video:generator', 'ghlink-video:result'].includes(node.type) ? (m.ghMode || 'video') : null;
        if (!kind) return [];
        const primary = kind === 'image' && Array.isArray(m.images) ? m.images.find(x => x.id === m.primaryImageId) || m.images.find(x => x.status === 'success') : null;
        const data = primary || m;
        const source = safeURL(data.remoteUrl) || safeURL(m.ghResultURL) || (/^data:image\/(png|jpeg|webp);base64,/.test(m.ghResultURL || '') ? m.ghResultURL : '') || data.content || '';
        const storageKey = data.storageKey || '';
        const identity = `${kind}:${storageKey || source || node.id}`;
        if (seen.has(identity)) return [];
        seen.add(identity);
        return [{ id: node.id, kind, title: node.title || kind, source, storageKey, mimeType: data.mimeType || m.mimeType || '', pluginBlobKey: m.ghBlobKey || '', identity }];
    });
}

// Read only the known media key referenced by an incoming node, using the host's existing stores.
export function readCanvasRecord(store, key) {
    if (!key || typeof indexedDB === 'undefined') return Promise.resolve(null);
    return new Promise(resolve => {
        const request = indexedDB.open('infinite-canvas');
        request.onupgradeneeded = () => request.transaction.abort(); // Never create or migrate the host database.
        request.onerror = () => resolve(null);
        request.onsuccess = () => {
            const db = request.result;
            if (!db.objectStoreNames.contains(store)) { db.close(); resolve(null); return; }
            const tx = db.transaction(store, 'readonly');
            const get = tx.objectStore(store).get(key);
            get.onsuccess = () => resolve(get.result ?? null);
            get.onerror = () => resolve(null);
            tx.oncomplete = tx.onabort = () => db.close();
        };
    });
}

export async function uploadReference(blob, item, signal, options = {}) {
    const runtimeConfig = options.runtimeConfig || window.__RUNTIME_CONFIG__ || {};
    const origin = options.origin || window.location.origin;
    if (!runtimeConfig.MEDIA_UPLOAD_ENDPOINT) throw new Error('此站点未启用统一媒体上传，请先上传素材并填写公共链接');
    const endpoint = new URL(runtimeConfig.MEDIA_UPLOAD_ENDPOINT, origin);
    if (endpoint.origin !== origin || !/^https?:$/.test(endpoint.protocol)) throw new Error('媒体上传入口必须与画布同源');
    endpoint.searchParams.set('returnFormat', 'full'); endpoint.searchParams.set('serverCompress', 'false');
    const form = new FormData();
    form.append('file', blob, item.title || 'reference');
    const response = await (options.fetcher || fetch)(endpoint.href, { method: 'POST', body: form, credentials: 'same-origin', redirect: 'error', signal: AbortSignal.any([signal, AbortSignal.timeout(120000)]) });
    const payload = await response.json().catch(() => null);
    const entry = Array.isArray(payload) ? payload[0] : null;
    if (!response.ok || typeof (entry?.publicUrl || entry?.src) !== 'string') throw new Error(`素材「${item.title}」上传失败（HTTP ${response.status}）`);
    const url = new URL(entry.publicUrl || entry.src, runtimeConfig.MEDIA_UPLOAD_BASE_URL || origin).href;
    if (!safeURL(url)) throw new Error('媒体上传未返回有效的公共链接');
    return url;
}

export async function prepareReferences(items, manual, signal, options = {}) {
    const read = options.read || readCanvasRecord;
    const upload = options.upload || uploadReference;
    const fetcher = options.fetcher || fetch;
    const cache = options.cache || new Map();
    const result = { images: [], videos: [], audios: [] };
    for (const item of items.filter(x => x.kind !== 'text')) {
        signal.throwIfAborted();
        options.onProgress?.(`读取${{ image: '图片', video: '视频', audio: '音频' }[item.kind]}：${item.title}`);
        let url = safeURL(item.source) || cache.get(item.identity);
        if (!url && item.storageKey) url = safeURL((await read('public_media_urls', item.storageKey))?.url);
        if (!url) {
            let blob = item.storageKey ? await read(item.kind === 'image' ? 'image_files' : 'media_files', item.storageKey) : null;
            if (!(blob instanceof Blob) && /^(blob:|data:(image|video|audio)\/)/.test(item.source)) {
                const response = await fetcher(item.source, { credentials: 'omit', signal });
                if (response.ok) blob = await response.blob();
            }
            if (!(blob instanceof Blob) || !blob.size) throw new Error(`无法读取连接素材「${item.title}」，请重新导入该素材后再生成`);
            if (blob.type && !blob.type.startsWith(`${item.kind}/`) && blob.type !== 'application/octet-stream') throw new Error(`连接素材「${item.title}」的文件类型与节点不一致`);
            const limit = options.parameters?.[`max_${item.kind}_size_mb`];
            if (limit > 0 && blob.size > limit * 1024 * 1024) throw new Error(`素材「${item.title}」超过模型 ${limit} MB 限制`);
            if (!blob.type || blob.type === 'application/octet-stream') blob = new Blob([blob], { type: item.mimeType || `${item.kind}/${item.kind === 'image' ? 'png' : item.kind === 'video' ? 'mp4' : 'mpeg'}` });
            options.onProgress?.(`上传参考素材：${item.title}`);
            url = await upload(blob, item, signal);
            signal.throwIfAborted();
            cache.set(item.identity, url);
        }
        options.onResolved?.(item, url);
        result[`${item.kind}s`].push(url);
    }
    for (const kind of Object.keys(result)) result[kind] = [...new Set([...result[kind], ...referenceURLs(manual[kind])])];
    return result;
}

export function imageResults(response) {
    const data = Array.isArray(response?.data) ? response.data : response?.data?.data;
    const urls = (Array.isArray(data) ? data : []).map(item => safeURL(item.url) || (typeof item.b64_json === 'string' && /^[A-Za-z0-9+/=\s]+$/.test(item.b64_json) ? `data:image/png;base64,${item.b64_json}` : '')).filter(Boolean);
    if (!urls.length) throw new Error('图片接口没有返回图片，请核对 GH 记录，不要直接重复提交');
    return urls;
}

// Read public result bytes without the GH key. A blob download avoids the
// cross-origin <a download> behavior that can navigate to the media website.
export async function readResultFile(url, mode = 'image', { fetcher = fetch, signal } = {}) {
    if (!safeURL(url) && !/^blob:/.test(url) && !/^data:image\/(png|jpeg|webp);base64,[A-Za-z0-9+/=\s]+$/.test(url)) throw new Error('结果地址无效，无法下载');
    const response = await fetcher(url, { credentials: 'omit', redirect: 'error', referrerPolicy: 'no-referrer', signal: signal ? AbortSignal.any([signal, AbortSignal.timeout(180000)]) : AbortSignal.timeout(180000) });
    if (!response.ok) throw new Error(`文件读取失败（HTTP ${response.status}），请稍后重试下载`);
    const blob = await response.blob();
    const type = blob.type.split(';')[0].toLowerCase();
    if (!blob.size || !(type.startsWith(`${mode}/`) || type === 'application/octet-stream')) throw new Error('返回的内容不是有效媒体文件，请稍后重试下载');
    const ext = { 'image/png': 'png', 'image/jpeg': 'jpg', 'image/webp': 'webp', 'image/gif': 'gif', 'image/avif': 'avif', 'video/mp4': 'mp4', 'video/webm': 'webm', 'video/quicktime': 'mov' }[type] || 'bin';
    return { blob, filename: `ghlink-${mode}.${ext}` };
}

export function priceLabel(model, resolution) {
    if (!model) return '价格将从 GH 实时读取';
    if (model.price_configured === false) return '尚未配置价格，不可生成';
    const unit = { request: '次', second: '秒' }[model.price_unit];
    const amount = model.video_resolution_prices?.[resolution] ?? model.model_price;
    if (!unit || amount === undefined || amount === null) return '按 GH 实际用量结算';
    return `${model.currency} ${amount} / ${unit}${model.price_scope === 'base' ? '（基础价，结算另含分组及参数倍率）' : ''}`;
}

function sleep(ms, signal) {
    return new Promise((resolve, reject) => {
        if (signal.aborted) return reject(new DOMException('Stopped', 'AbortError'));
        const stopped = () => { clearTimeout(timer); reject(new DOMException('Stopped', 'AbortError')); };
        const timer = setTimeout(() => { signal.removeEventListener('abort', stopped); resolve(); }, ms);
        signal.addEventListener('abort', stopped, { once: true });
    });
}

export async function pollTask(key, id, signal, onState, options = {}) {
    const call = options.call || api;
    const wait = options.wait || sleep;
    const deadline = Date.now() + (options.timeoutMs ?? 30 * 60 * 1000);
    while (!signal.aborted && Date.now() < deadline) {
        let task;
        try { task = unwrap(await call(key, `/v1/videos/${encodeURIComponent(id)}`, { signal })); }
        catch (error) {
            if (signal.aborted || (error.status && error.status !== 429 && error.status < 500)) throw error;
            onState({ status: '查询暂时失败，将继续重试' });
            await wait(Math.max(40000, error.retryMs || 0), signal);
            continue;
        }
        onState(task);
        if (FAILED.has(statusOf(task))) throw Object.assign(new Error(publicError(task.error?.message || task.message || task.fail_reason || '视频生成失败', key)), { terminal: true });
        if (DONE.has(statusOf(task))) return task;
        await wait(20000, signal);
    }
    throw new Error('查询已暂停，任务 ID 已保留。可继续查询，无需再次提交。');
}

export function resultNodeOps(source, nodes, id, mode) {
    const width = 360, height = 300;
    const x = (source.position?.x || 0) + (source.width || 440) + 80;
    let y = source.position?.y || 0;
    while (nodes.some(n => x < n.position.x + n.width + 24 && x + width + 24 > n.position.x && y < n.position.y + n.height + 24 && y + height + 24 > n.position.y)) y += height + 40;
    return [
        { type: 'add_node', id, nodeType: 'ghlink-video:result', title: mode === 'image' ? '图片生成结果' : '视频生成结果', position: { x, y }, width, height, metadata: { ghMode: mode, ghSourceNodeId: source.id } },
        { type: 'connect_nodes', fromNodeId: source.id, toNodeId: id },
    ];
}

export default function createPlugin(runtime) {
    const { React: R } = runtime;
    const h = R.createElement;
    // Credentials stay in memory; never written to node metadata, plugin storage or install URLs.
    let sessionKey = '';
    const uploadedReferences = new Map();
    const resultURLs = new Map();
    const downloadURLs = new Map();
    function saveResultFile({ blob, filename }) {
        const url = URL.createObjectURL(blob), link = document.createElement('a');
        link.href = url; link.download = filename; link.hidden = true;
        document.body.append(link);
        try { link.click(); }
        finally {
            link.remove();
            downloadURLs.set(url, setTimeout(() => { URL.revokeObjectURL(url); downloadURLs.delete(url); }, 30000));
        }
    }
    // Canvas nodes can unmount while their requests are still running. Keep each
    // node's task, lock and UI state for the lifetime of this plugin instance.
    const sessions = new Map();
    const notify = session => { for (const listener of session.listeners) listener(); };
    const preventReload = event => { event.preventDefault(); event.returnValue = ''; };
    function syncReloadGuard() {
        if (typeof window === 'undefined') return;
        if ([...sessions.values()].some(session => session.generation && session.busy)) window.addEventListener('beforeunload', preventReload);
        else window.removeEventListener('beforeunload', preventReload);
    }
    function PromptEditor({ value, bindings, items, disabled, theme, onChange }) {
        const input = R.useRef(null);
        const list = R.useRef(null);
        const composing = R.useRef(false);
        const [mention, setMention] = R.useState(null);
        const [active, setActive] = R.useState(0);
        const listId = R.useId();
        const options = mentionOptions(items, bindings);
        const query = mention?.query.toLowerCase() || '';
        const candidates = options.filter(x => `${x.token} ${x.title} ${x.kind}`.toLowerCase().includes(query));
        const selected = Math.min(active, Math.max(0, candidates.length - 1));
        const tokens = [...new Set([...value.matchAll(mentionPattern())].map(x => x[0]))];
        R.useEffect(() => {
            const menu = list.current, option = menu?.children[selected];
            if (!menu || !option) return;
            const top = option.offsetTop - menu.scrollTop;
            const bottom = top + option.offsetHeight;
            if (top < 0) menu.scrollTop += top;
            else if (bottom > menu.clientHeight) menu.scrollTop += bottom - menu.clientHeight;
        }, [selected, mention?.query]);
        function sync(element) {
            setMention(element.selectionStart === element.selectionEnd ? mentionAt(element.value, element.selectionStart) : null);
            setActive(0);
        }
        function insert(item, fromMenu = false) {
            const element = input.current;
            if (!element || disabled) return;
            const range = fromMenu && mention ? mention : { start: element.selectionStart, end: element.selectionEnd };
            const next = insertMention(value, range, item, bindings);
            onChange(next.prompt, next.bindings);
            setMention(null);
            requestAnimationFrame(() => { element.focus(); element.setSelectionRange(next.cursor, next.cursor); });
        }
        const small = { fontSize: 11, color: theme.node.muted };
        return h('div', { style: { display: 'grid', gap: 6 } },
            h('label', { style: { display: 'grid', gap: 4 } }, h('span', { style: small }, '提示词'),
                h('textarea', {
                    ref: input, value, disabled, placeholder: '描述画面、动作；输入 @ 选择参考素材',
                    'aria-autocomplete': 'list', 'aria-controls': mention ? listId : undefined, 'aria-expanded': !!mention,
                    'aria-activedescendant': mention && candidates.length ? `${listId}-${selected}` : undefined,
                    style: { width: '100%', boxSizing: 'border-box', minHeight: 100, padding: '9px 10px', borderRadius: 7, border: `1px solid ${theme.node.stroke}`, color: theme.node.text, background: theme.toolbar.panel, fontSize: 12, lineHeight: 1.7, resize: 'vertical' },
                    onChange: e => { onChange(e.target.value, bindings); if (!composing.current) sync(e.target); },
                    onClick: e => sync(e.target), onFocus: e => sync(e.target), onBlur: () => setMention(null),
                    onCompositionStart: () => { composing.current = true; setMention(null); },
                    onCompositionEnd: e => { composing.current = false; sync(e.target); },
                    onKeyUp: e => { if (!composing.current && ['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(e.key)) sync(e.target); },
                    onKeyDown: e => {
                        if (composing.current || e.nativeEvent.isComposing || e.keyCode === 229 || !mention) return;
                        if (e.key === 'Escape') { e.preventDefault(); setMention(null); }
                        else if (candidates.length && ['ArrowDown', 'ArrowUp'].includes(e.key)) { e.preventDefault(); setActive((selected + (e.key === 'ArrowDown' ? 1 : candidates.length - 1)) % candidates.length); }
                        else if (candidates.length && e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); insert(candidates[selected], true); }
                    },
                })),
            mention ? h('div', { style: { border: `1px solid ${theme.node.stroke}`, borderRadius: 8, padding: 6, background: theme.toolbar.panel } },
                h('div', { style: { ...small, padding: '3px 5px 7px' } }, '选择参考素材 · ↑↓ 选择 · Enter 插入'),
                h('div', { ref: list, id: listId, role: 'listbox', 'aria-label': '参考素材', style: { position: 'relative', maxHeight: 180, overflowY: 'auto', display: 'grid', gap: 3 } },
                    candidates.length ? candidates.map((item, index) => h('button', {
                        key: item.id, id: `${listId}-${index}`, role: 'option', type: 'button', 'aria-selected': index === selected,
                        onMouseDown: e => e.preventDefault(), onClick: () => insert(item, true),
                        style: { display: 'flex', alignItems: 'center', gap: 8, textAlign: 'left', border: index === selected ? '1px solid #5686f7' : '1px solid transparent', borderRadius: 6, padding: '7px 8px', background: index === selected ? '#5686f718' : 'transparent', color: theme.node.text, cursor: 'pointer' },
                    }, h('span', { style: { fontSize: 18 } }, { image: '▧', video: '▷', audio: '♫' }[item.kind]), h('span', { style: { minWidth: 0, display: 'grid', gap: 3 } }, h('strong', { style: { fontSize: 12 } }, `${item.token} · ${item.title}`), h('span', { style: small }, `已连接${MENTION_NAMES[item.kind]}`)))) : h('div', { style: { ...small, padding: 8 } }, options.length ? '没有匹配素材，试试文件名或素材类型。' : '请先把图片、视频或音频节点连入插件左侧接口。')),
            ) : null,
            tokens.length ? h('div', { 'aria-label': '提示词中的素材引用', style: { display: 'flex', flexWrap: 'wrap', gap: 5 } }, ...tokens.map(token => {
                const bound = bindings.find(x => x.token === token);
                const item = bound && options.find(x => x.token === token);
                const missing = bound && !item;
                return h('span', { key: token, title: bound?.title || '手动引用编号', style: { border: `1px solid ${missing ? '#df7373' : theme.node.stroke}`, borderRadius: 5, padding: '3px 6px', fontSize: 11, color: missing ? '#cf4848' : theme.node.text, overflowWrap: 'anywhere' } }, `${token}${bound ? ` · ${bound.title}` : ''}${missing ? '（已断开）' : ''}`);
            })) : null,
            h('span', { style: { ...small, lineHeight: 1.6 } }, '连线素材会一起提交；输入 @ 可在提示词中指定各素材的用途。'),
        );
    }
    function ImageViewer({ src, theme, downloading, onDownload, onClose, downloadMessage }) {
        const dialog = R.useRef(null), stage = R.useRef(null);
        const [size, setSize] = R.useState({ width: 0, height: 0 });
        const [space, setSpace] = R.useState({ width: 0, height: 0 });
        const [zoom, setZoom] = R.useState(null);
        const [failed, setFailed] = R.useState(false);
        const titleId = R.useId();
        R.useEffect(() => {
            if (!dialog.current.open) dialog.current.showModal();
            const element = stage.current;
            const measure = () => setSpace({ width: element.clientWidth, height: element.clientHeight });
            measure(); const observer = new ResizeObserver(measure); observer.observe(element);
            return () => observer.disconnect();
        }, []);
        const fit = size.width && space.width ? Math.min((space.width - 36) / size.width, (space.height - 36) / size.height, 1) : 1;
        const scale = zoom ?? Math.max(.01, fit);
        const width = size.width * scale, height = size.height * scale;
        const close = () => { dialog.current?.close(); onClose(); };
        const control = (text, action, label = text, disabled = false) => h('button', { type: 'button', className: 'ghlink-action ghlink-action--secondary', 'aria-label': label, title: label, disabled, onClick: action }, text);
        return h('dialog', {
            ref: dialog, className: 'ghlink-viewer', 'aria-labelledby': titleId,
            style: { '--gh-button-panel': theme.toolbar.panel, '--gh-button-text': theme.node.text, '--gh-button-border': theme.node.stroke, '--gh-button-muted': theme.node.muted },
            onPointerDown: e => e.stopPropagation(), onMouseDown: e => e.stopPropagation(), onWheel: e => e.stopPropagation(), onKeyDown: e => e.stopPropagation(),
            onCancel: e => { e.preventDefault(); close(); }, onClose,
            onClick: e => { const rect = e.currentTarget.getBoundingClientRect(); if (e.target === e.currentTarget && (e.clientX < rect.left || e.clientX > rect.right || e.clientY < rect.top || e.clientY > rect.bottom)) close(); },
        },
            h('header', { className: 'ghlink-viewer-header' },
                h('div', null, h('strong', { id: titleId }, '图片预览'), h('div', { className: 'ghlink-media-meta' }, size.width ? `${size.width} × ${size.height}` : '正在加载图片…')),
                control('×', close, '关闭图片预览'),
                h('div', { className: 'ghlink-viewer-tools' },
                    control('−', () => setZoom(Math.max(.01, scale / 1.25)), '缩小图片', !size.width),
                    h('span', { className: 'ghlink-viewer-scale', 'aria-live': 'polite' }, `${Math.round(scale * 100)}%`),
                    control('+', () => setZoom(Math.min(8, scale * 1.25)), '放大图片', !size.width),
                    control('适应窗口', () => setZoom(null)), control('原始尺寸', () => setZoom(1), '原始尺寸', !size.width),
                    control(downloading ? '下载中…' : '下载图片', onDownload, '下载图片', downloading))),
            h('div', { ref: stage, className: 'ghlink-viewer-stage' },
                failed ? h('div', { role: 'alert', className: 'ghlink-media-empty' }, '图片加载失败，请关闭后重新查看。') : h('div', { className: 'ghlink-viewer-image-wrap', style: { width: width ? Math.ceil(width + 32) : '100%', height: height ? Math.ceil(height + 32) : '100%' } },
                    h('img', { src, alt: '生成图片的大图预览', draggable: false, onLoad: e => setSize({ width: e.currentTarget.naturalWidth, height: e.currentTarget.naturalHeight }), onError: () => setFailed(true), style: { width: width || undefined, height: height || undefined, maxWidth: size.width ? 'none' : '100%' } }))),
            h('footer', { className: 'ghlink-viewer-footer', role: 'status' }, downloadMessage || '放大后可滚动查看细节 · Esc 关闭'),
        );
    }
    function Result({ ctx }) {
        const m = ctx.node.metadata || {};
        const progress = m.ghProgress || { phase: 'idle', message: '等待生成' };
        const [localURL, setLocalURL] = R.useState(resultURLs.get(ctx.node.id) || '');
        const [elapsed, setElapsed] = R.useState(0);
        const [viewer, setViewer] = R.useState(false);
        const [dimensions, setDimensions] = R.useState({ src: '', text: '' });
        const [failed, setFailed] = R.useState('');
        const [retry, setRetry] = R.useState(0);
        const [downloading, setDownloading] = R.useState(false);
        const [downloadMessage, setDownloadMessage] = R.useState('');
        const downloadController = R.useRef(null);
        const pending = ['preparing', 'submitting', 'generating', 'loading'].includes(progress.phase);
        R.useEffect(() => {
            let disposed = false;
            if (m.ghBlobKey) ctx.storage.get(m.ghBlobKey).then(blob => {
                if (disposed || !(blob instanceof Blob)) return;
                let url = resultURLs.get(ctx.node.id);
                if (!url) { url = URL.createObjectURL(blob); resultURLs.set(ctx.node.id, url); }
                setLocalURL(url);
            }).catch(() => {});
            return () => { disposed = true; };
        }, [m.ghBlobKey]);
        R.useEffect(() => {
            if (!pending) return;
            const tick = () => setElapsed(Math.max(0, Math.floor((Date.now() - progress.started) / 1000)));
            tick(); const timer = setInterval(tick, 1000); return () => clearInterval(timer);
        }, [pending, progress.started]);
        R.useEffect(() => () => downloadController.current?.abort(), []);
        const titles = { preparing: '正在准备参考素材', submitting: '正在提交请求', generating: `${m.ghMode === 'image' ? '图片' : '视频'}正在生成`, loading: '正在读取结果', completed: '生成完成', error: '生成未完成', paused: '查询已暂停' };
        const preview = m.ghResultURL || localURL;
        R.useEffect(() => { setDownloadMessage(''); setViewer(false); }, [preview]);
        const image = m.ghMode === 'image', ready = preview && progress.phase === 'completed';
        async function download() {
            if (downloadController.current) return;
            const controller = new AbortController(); downloadController.current = controller;
            setDownloading(true); setDownloadMessage('');
            try { saveResultFile(await readResultFile(preview, image ? 'image' : 'video', { signal: controller.signal })); setDownloadMessage('已开始下载'); }
            catch (error) { if (!controller.signal.aborted) setDownloadMessage(`下载失败：${publicError(error)}。请重试下载，结果仍可在站内查看。`); }
            finally { downloadController.current = null; if (!controller.signal.aborted) setDownloading(false); }
        }
        const action = (text, onClick, disabled = false) => h('button', { type: 'button', className: 'ghlink-action ghlink-action--secondary', disabled, onClick }, text);
        return h('div', { className: 'ghlink-result', onPointerDown: e => e.stopPropagation(), onMouseDown: e => e.stopPropagation(), onDoubleClick: e => e.stopPropagation(), onWheel: e => e.stopPropagation(), style: { '--gh-button-panel': ctx.theme.toolbar.panel, '--gh-button-text': ctx.theme.node.text, '--gh-button-border': ctx.theme.node.stroke, '--gh-button-muted': ctx.theme.node.muted } },
            ready ? h('div', { className: 'ghlink-result-media' }, image ? (failed === preview ? h('div', { role: 'alert', className: 'ghlink-media-empty' }, '图片暂未加载', action('重新加载', () => { setFailed(''); setRetry(value => value + 1); })) : h('button', { type: 'button', className: 'ghlink-image-preview', 'aria-label': '查看生成图片大图', onClick: () => setViewer(true) },
                h('img', { key: `${preview}:${retry}`, src: preview, alt: '生成的图片', draggable: false, onLoad: e => setDimensions({ src: preview, text: `${e.currentTarget.naturalWidth} × ${e.currentTarget.naturalHeight}` }), onError: () => setFailed(preview) }), h('span', { className: 'ghlink-image-hint' }, '点击查看大图'))) : h('video', { src: preview, controls: true, style: { width: '100%', height: '100%', objectFit: 'contain' } })) : h('div', { style: { flex: 1, minHeight: 70, display: 'grid', placeContent: 'center', textAlign: 'center', gap: 12 } },
                h('span', { style: { fontSize: 28, color: ctx.theme.node.muted } }, m.ghMode === 'image' ? '▧' : '▷'),
                h('strong', { style: { fontSize: 15 } }, titles[progress.phase] || '等待生成'),
                pending ? h('progress', { max: 100, ...(Number.isFinite(progress.percent) ? { value: progress.percent } : {}), 'aria-label': '生成进度', style: { width: 220, height: 6, accentColor: '#5285f6' } }) : null,
                pending ? h('span', { style: { fontSize: 12, color: ctx.theme.node.muted } }, `${Math.floor(elapsed / 60)} 分 ${elapsed % 60} 秒${Number.isFinite(progress.percent) ? ` · ${progress.percent}%` : ''}`) : null),
            ready ? h('div', { className: 'ghlink-result-caption' }, h('strong', null, '生成完成'), h('span', { className: 'ghlink-media-meta' }, image ? dimensions.src === preview ? dimensions.text : '' : '视频')) : null,
            ready ? h('div', { className: 'ghlink-actions' }, image ? action('查看大图', () => setViewer(true)) : null, action(downloading ? '下载中…' : image ? '下载图片' : '下载视频', download, downloading)) : null,
            h('div', { role: 'status', className: 'ghlink-media-meta' }, downloadMessage || (ready ? '' : progress.message)),
            progress.taskId ? h('details', { onPointerDown: e => e.stopPropagation(), style: { fontSize: 11, color: ctx.theme.node.muted, overflowWrap: 'anywhere' } }, h('summary', null, '任务信息'), progress.taskId, h('div', null, '刷新或停止等待后，可在原插件节点继续查询。')) : null,
            ready && image && viewer ? h(ImageViewer, { src: preview, theme: ctx.theme, downloading, onDownload: download, downloadMessage, onClose: () => setViewer(false) }) : null,
        );
    }
    function Content({ ctx }) {
        const m = ctx.node.metadata || {};
        let session = sessions.get(ctx.node.id);
        if (!session) {
            session = { key: sessionKey, models: [], busy: false, generation: false, message: '填写 API Key，读取你有权限使用的模型', listeners: new Set(),
                progressRef: { current: { phase: 'idle', message: '', started: 0 } }, resultId: { current: m.ghResultNodeId || '' },
                ctxRef: { current: ctx }, controller: { current: null }, lock: { current: false } };
            sessions.set(ctx.node.id, session);
        }
        const { progressRef, resultId, ctxRef, controller, lock } = session;
        ctxRef.current = ctx;
        const [, redraw] = R.useState(0);
        const { key, models, busy, message } = session;
        const setKey = value => { session.key = value; notify(session); };
        const setModels = value => { session.models = value; notify(session); };
        const setBusy = value => { session.busy = value; notify(session); syncReloadGuard(); };
        const setMessage = value => { session.message = value; notify(session); };
        const [connected, setConnected] = R.useState(() => connectedResources(ctx.getUpstream()));
        const mode = m.ghMode || 'video';
        const visibleModels = models.filter(item => item.kind === mode);
        const model = visibleModels.find(item => item.id === m.ghModel);
        const p = model?.parameters || {};
        const labels = new Map(mentionOptions(connected, m.ghMentionBindings || []).map(item => [item.id, item.token]));
        R.useEffect(() => {
            const listener = () => redraw(value => value + 1);
            session.listeners.add(listener); listener();
            // Detaching the view must not cancel an accepted or in-flight task.
            return () => { session.listeners.delete(listener); };
        }, [session]);
        R.useEffect(() => {
            const update = () => { const next = connectedResources(ctxRef.current.getUpstream()); setConnected(previous => JSON.stringify(previous) === JSON.stringify(next) ? previous : next); };
            update(); const timer = setInterval(update, 800); return () => clearInterval(timer);
        }, []);
        function patch(value) {
            if (ctxRef.current.getNode(ctx.node.id)) ctxRef.current.applyOps([{ type: 'update_node', id: ctx.node.id, metadata: value }]);
        }
        function updateResult(metadata) {
            if (ctxRef.current.getNode(ctx.node.id) && resultId.current && ctxRef.current.getNode(resultId.current)) ctxRef.current.applyOps([{ type: 'update_node', id: resultId.current, metadata }]);
        }
        function report(message, phase, extra = {}) {
            setMessage(message);
            progressRef.current = { ...progressRef.current, phase, message, percent: undefined, ...extra };
            updateResult({ ghProgress: progressRef.current });
        }
        function ensureResult() {
            const context = ctxRef.current;
            if (!context.getNode(ctx.node.id)) throw new Error('原插件节点已删除');
            if (!resultId.current || !context.getNode(resultId.current)) {
                resultId.current = `gh-result-${crypto.randomUUID()}`;
                context.applyOps(resultNodeOps(context.node, context.getNodes(), resultId.current, mode));
                patch({ ghResultNodeId: resultId.current });
            }
            progressRef.current = { phase: 'preparing', message: '正在读取连接素材与模型参数…', started: Date.now() };
            updateResult({ ghProgress: progressRef.current });
        }
        async function run(action, generation = false) {
            if (lock.current) return;
            session.generation = generation;
            lock.current = true; setBusy(true);

            const aborter = new AbortController(); controller.current = aborter;
            // Task lifetime is tied to the canvas node, not its mounted view.
            const timer = setInterval(() => { if (!ctxRef.current.getNode(ctx.node.id)) aborter.abort(); }, 800);
            try { if (generation) ensureResult(); await action(aborter.signal); }
            catch (error) {
                const text = aborter.signal.aborted ? '已停止本页等待；已提交的 GH 任务不会被取消，请通过任务 ID 或 GH 记录核对。' : publicError(error, key);
                if (generation) report(text, aborter.signal.aborted ? 'paused' : 'error'); else setMessage(text);
            }
            finally { clearInterval(timer); controller.current = null; lock.current = false; setBusy(false); }
        }
        async function refresh(signal) {
            setModels([]);
            const [list, prices] = await Promise.all([api(key, '/v1/models', { signal }), api(key, '/v1/pricing?currency=CNY', { signal })]);
            const available = [...catalog(list, prices), ...catalog(list, prices, 'image')];
            setModels(available);
            if (!available.some(item => item.id === m.ghModel && item.kind === mode)) choose(available.find(item => item.kind === mode));
            setMessage(`已读取 ${available.filter(x => x.kind === 'video').length} 个视频模型、${available.filter(x => x.kind === 'image').length} 个图片模型`);
        }
        function choose(next) {
            const spec = next?.parameters || {};
            patch({ ghModel: next?.id || '', ghSeconds: String(spec.durations?.[0] || ''), ghRatio: spec.ratios?.[0] || '', ghResolution: spec.resolutions?.[0] || '' });
        }
        async function finish(id, signal) {
            report('视频正在生成，请稍候…', 'generating', { taskId: id });
            let task;
            try { task = await pollTask(key, id, signal, value => {
                const percent = Number.parseFloat(value.progress);
                report(({ queued: '任务排队中，请稍候…', pending: '任务排队中，请稍候…', processing: '视频正在生成，请稍候…', running: '视频正在生成，请稍候…', completed: '正在读取生成结果…', succeeded: '正在读取生成结果…' })[value.status || value.state] || '视频正在生成，请稍候…', 'generating', { taskId: id, percent: Number.isFinite(percent) && percent >= 0 && percent <= 100 ? percent : undefined });
            }); }
            catch (error) { if (error.terminal) patch({ ghPhase: 'failed' }); throw error; }
            patch({ ghPhase: 'completed' });
            report('生成已完成，正在读取视频结果…', 'loading', { taskId: id });
            const resultURL = safeURL(task.video_url || task.url || task.result_url || task.output?.url || task.video?.url || task.data?.url);
            // The authenticated content endpoint is fetched as a blob, never used as a key-bearing video URL.
            if (!resultURL || new URL(resultURL).origin === ORIGIN) {
                const blob = await api(key, `/v1/videos/${encodeURIComponent(id)}/content`, { signal, binary: true });
                const blobKey = `result:${resultId.current}`;
                await ctxRef.current.storage.set(blobKey, blob);
                signal.throwIfAborted();
                if (resultURLs.has(resultId.current)) URL.revokeObjectURL(resultURLs.get(resultId.current));
                resultURLs.set(resultId.current, URL.createObjectURL(blob));
                updateResult({ ghBlobKey: blobKey, ghResultURL: '' });
                patch({ ghResultURL: '', ghBlobKey: blobKey });
            } else { patch({ ghResultURL: resultURL }); updateResult({ ghResultURL: resultURL }); }
            report('视频已完成，可预览或下载。任务 ID 已保留。', 'completed', { taskId: id });
        }
        async function submit(signal) {
            if (m.ghTaskId || m.ghPhase) throw new Error('已有任务，请继续查询或先确认任务状态');
            if (!key.trim()) throw new Error('请填写 API Key');
            const inputs = await Promise.all(connectedResources(ctxRef.current.getUpstream()).map(async item => {
                if (item.kind === 'text' || safeURL(item.source)) return item;
                if (item.pluginBlobKey) {
                    let url = resultURLs.get(item.id);
                    if (!url) { const blob = await ctxRef.current.storage.get(item.pluginBlobKey); if (blob instanceof Blob) { url = URL.createObjectURL(blob); resultURLs.set(item.id, url); } }
                    if (url) return { ...item, source: url };
                }
                const known = uploadedReferences.get(item.identity) || (item.storageKey ? (await readCanvasRecord('public_media_urls', item.storageKey))?.url : '');
                return safeURL(known) ? { ...item, source: known } : item;
            }));
            const manual = { images: m.ghImages, videos: m.ghVideos, audios: m.ghAudios };
            const values = { prompt: m.ghPrompt || inputs.filter(x => x.kind === 'text').map(x => x.source).join('\n'), seconds: m.ghSeconds, ratio: m.ghRatio, resolution: m.ghResolution };
            const bindings = m.ghMentionBindings || [];
            const previewURLs = new Map(inputs.filter(x => x.kind !== 'text').map(x => [x.id, safeURL(x.source) || `https://local.invalid/${encodeURIComponent(x.identity)}`]));
            const previewRefs = {};
            for (const [kind, plural] of [['image', 'images'], ['video', 'videos'], ['audio', 'audios']]) previewRefs[plural] = [...new Set([...inputs.filter(x => x.kind === kind).map(x => previewURLs.get(x.id)), ...referenceURLs(manual[plural])])];
            const previewValues = { ...values, ...previewRefs, prompt: compileMentions(values.prompt, bindings, inputs, previewRefs, previewURLs) };
            if (mode === 'image') buildImageRequest(model, previewValues);
            else buildPayload(model, previewValues); // Validate bindings and combined limits before uploading files.
            const resolvedURLs = new Map();
            const refs = await prepareReferences(inputs, manual, signal, { cache: uploadedReferences, parameters: p, onResolved: (item, url) => resolvedURLs.set(item.id, url), onProgress: text => report(text, 'preparing') });
            const finalValues = { ...values, ...refs, prompt: compileMentions(values.prompt, bindings, inputs, refs, resolvedURLs) };
            // The catalog's image route is automatic. Do not reuse obsolete
            // per-node protocol overrides or switch routes after a failure.
            const request = mode === 'image' ? buildImageRequest(model, finalValues) : { path: '/v1/videos', body: buildPayload(model, finalValues) };
            const summary = { path: request.path, model: model.id, images: refs.images.length, videos: refs.videos.length, audios: refs.audios.length };
            patch({ ghLastRequest: summary }); updateResult({ ghLastRequest: summary });
            signal.throwIfAborted();
            patch({ ghPhase: 'submitting', ghResultURL: '' });
            report(`正在提交${mode === 'image' ? '图片' : '视频'}生成请求…`, 'submitting');
            let response;
            try {
                if (mode === 'image') {
                    report('图片正在生成，请稍候…', 'generating');
                    response = await api(key, request.path, { body: request.body, signal, timeoutMs: 300000 });
                    const outputs = imageRequestResults(response, request.path, refs.images);
                    patch({ ghPhase: 'completed', ghResultURL: outputs[0] });
                    updateResult({ ghResultURL: outputs[0] });
                    report('图片生成完成，可预览或下载。', 'completed');
                    return;
                }
                response = unwrap(await api(key, request.path, { body: request.body, signal }));
            }
            catch (error) {
                patch({ ghPhase: 'uncertain' });
                throw new Error(`${publicError(error, key)}。提交结果未确认，请先到 GH 任务记录核对，勿重复提交。`);
            }
            const id = response?.id || response?.task_id;
            if (!id) { patch({ ghPhase: 'uncertain' }); throw new Error('未收到任务 ID，请先到 GH 任务记录核对，勿重复提交'); }
            patch({ ghTaskId: String(id), ghPhase: 'pending' });
            await finish(String(id), signal);
        }
        function reset() {
            if (!['completed', 'failed'].includes(m.ghPhase) && !window.confirm('请先在 GH 确认旧请求未受理或任务已经结束。新提交可能再次计费，确定重置？')) return;
            patch({ ghTaskId: '', ghPhase: '', ghResultURL: '', ghResultNodeId: '', ghBlobKey: '', ghLastRequest: null }); resultId.current = '';
            setMessage('可创建新任务');
        }
        const inputStyle = { width: '100%', minWidth: 0, boxSizing: 'border-box', padding: '7px 9px', borderRadius: 6, border: `1px solid ${ctx.theme.node.stroke}`, color: ctx.theme.node.text, background: ctx.theme.toolbar.panel, fontSize: 12 };
        const label = (text, child) => h('label', { style: { display: 'grid', gap: 4 } }, h('span', { style: { fontSize: 11, color: ctx.theme.node.muted } }, text), child);
        const button = (text, action, disabled = false, variant = 'plain') => h('button', {
            type: 'button', onClick: action, disabled,
            className: variant === 'plain' ? undefined : `ghlink-action ghlink-action--${variant}`,
            style: variant === 'plain' ? { background: 'transparent', color: ctx.theme.node.text, padding: '6px 9px', border: 'none', cursor: disabled ? 'default' : 'pointer', opacity: disabled ? .45 : 1, fontSize: 12 } : {
                '--gh-button-panel': ctx.theme.toolbar.panel, '--gh-button-text': ctx.theme.node.text,
                '--gh-button-border': ctx.theme.node.stroke, '--gh-button-muted': ctx.theme.node.muted,
            },
        }, text);
        const input = (name, placeholder, type = 'text') => h('input', { type, style: inputStyle, value: m[name] || '', placeholder, disabled: busy, onChange: e => patch({ [name]: e.target.value }) });
        const select = (name, title, values) => label(title, values.length ? h('select', { style: inputStyle, disabled: busy, value: m[name] || '', onChange: e => patch({ [name]: e.target.value }) }, ...values.map(value => h('option', { key: value, value }, value))) : input(name, '未公布范围，请填写', name === 'ghSeconds' ? 'number' : 'text'));
        return h('div', { 'data-canvas-no-zoom': true, onPointerDown: e => e.stopPropagation(), onMouseDown: e => e.stopPropagation(), onWheel: e => e.stopPropagation(), style: { height: '100%', width: '100%', boxSizing: 'border-box', overflow: 'auto', padding: 16, display: 'flex', flexDirection: 'column', gap: 10, color: ctx.theme.node.text } },
            h('div', { style: { fontSize: 16, fontWeight: 600 } }, 'GHLINK 视频 / 图片'),
            h('div', { style: { fontSize: 11, color: ctx.theme.node.muted } }, 'ghlink.top · Key 仅保留在本页内存，刷新后需重填'),
            label('GH API Key', h('input', { type: 'password', autoComplete: 'off', style: inputStyle, value: key, disabled: busy, placeholder: '填写你在 GH 创建的 API Key', onChange: e => { setKey(e.target.value); sessionKey = e.target.value; setModels([]); } })),
            button(busy ? '处理中…' : '连接并刷新模型', () => run(refresh), busy || !key.trim()),
            label('生成类型', h('select', { style: inputStyle, value: mode, disabled: busy || !!m.ghPhase || !!m.ghTaskId, onChange: e => { const kind = e.target.value; patch({ ghMode: kind }); choose(models.find(item => item.kind === kind)); } }, h('option', { value: 'video' }, '视频'), h('option', { value: 'image' }, '图片'))),
            label(mode === 'image' ? '图片模型' : '视频模型', h('select', { style: inputStyle, value: m.ghModel || '', disabled: busy || !visibleModels.length, onChange: e => choose(visibleModels.find(item => item.id === e.target.value)) }, h('option', { value: '' }, '刷新后选择模型'), ...visibleModels.map(item => h('option', { key: item.id, value: item.id, disabled: item.price_configured === false }, item.id)))),
            h('div', { style: { color: ctx.theme.node.muted, fontSize: 11, lineHeight: 1.5 } }, priceLabel(model, m.ghResolution)),
            h(PromptEditor, { value: m.ghPrompt || '', bindings: m.ghMentionBindings || [], items: connected, disabled: busy, theme: ctx.theme, onChange: (prompt, bindings) => patch({ ghPrompt: prompt, ghMentionBindings: bindings }) }),
            h('div', { style: { display: 'grid', gridTemplateColumns: `repeat(${mode === 'image' ? 2 : 3}, minmax(0, 1fr))`, gap: 8 } }, mode === 'image' ? null : select('ghSeconds', '时长 / 秒', p.durations || []), p.omit_ratio || (mode === 'image' && !p.ratio_field) ? null : select('ghRatio', '画面比例', strings(p.ratios)), select('ghResolution', mode === 'image' && !p.resolution_field ? '图片尺寸（可选）' : '分辨率', strings(p.resolutions))),
            h('div', { 'aria-label': '已连接素材', style: { display: 'grid', gap: 5, fontSize: 12, lineHeight: 1.6 } }, h('strong', null, `已连接素材 · ${connected.length} 个`), connected.length ? connected.map((item, index) => h('div', { key: item.id, style: { overflowWrap: 'anywhere', color: ctx.theme.node.muted } }, `${labels.get(item.id) || '文本'} · ${item.title}${safeURL(item.source) ? '' : item.kind === 'text' ? '（提示词为空时使用）' : '（生成时读取本地素材）'}`)) : h('span', { style: { color: ctx.theme.node.muted } }, '将图片、视频或音频节点连入左侧接口，即可自动读取。')),
            h('details', null, h('summary', { style: { fontSize: 12, cursor: 'pointer' } }, '补充参考素材链接'), h('div', { style: { display: 'grid', gap: 8, marginTop: 8 } }, ...[['ghImages', '图片', 'max_images'], ['ghVideos', '视频', 'max_videos'], ['ghAudios', '音频', 'max_audios']].map(([name, title, limit]) => label(`${title}${!p.capabilities_unknown && p[limit] !== undefined ? ` · 最多 ${p[limit]} 个（含连接素材）` : ' · 数量限制未公布'}`, h('textarea', { style: inputStyle, rows: 2, disabled: busy || (!p.capabilities_unknown && p[limit] === 0), value: m[name] || '', placeholder: '连接素材会自动读取；此处可额外补充链接', onChange: e => patch({ [name]: e.target.value }) }))))),
            h('div', { className: 'ghlink-actions' },
                button(mode === 'image' ? '生成图片' : '生成视频', () => run(submit, true), busy || !model || !key.trim() || !!m.ghTaskId || !!m.ghPhase || model?.price_configured === false, 'primary'),
                mode === 'video' ? button('继续查询', () => run(signal => finish(m.ghTaskId, signal), true), busy || !m.ghTaskId || !key.trim(), 'secondary') : null,
                busy ? button('停止本页等待', () => controller.current?.abort(), false, 'secondary') : (m.ghTaskId || m.ghPhase) ? button('新任务', reset, false, 'secondary') : null,
                m.ghResultNodeId ? button('选中结果节点', () => ctx.applyOps([{ type: 'select_nodes', ids: [m.ghResultNodeId] }]), false, 'secondary') : null),
            mode === 'video' ? label('任务 ID（用于恢复查询）', input('ghTaskId', '提交后自动保存，也可粘贴已有 GH 任务 ID')) : null,
            h('div', { role: 'status', style: { fontSize: 12, lineHeight: 1.6, overflowWrap: 'anywhere' } }, message),
            m.ghLastRequest ? h('details', null, h('summary', { style: { fontSize: 12, cursor: 'pointer' } }, '生成信息'), h('div', { style: { fontSize: 11, lineHeight: 1.7, overflowWrap: 'anywhere' } }, m.ghLastRequest.model, h('br'), `参考图片 ${m.ghLastRequest.images} 张、视频 ${m.ghLastRequest.videos} 个、音频 ${m.ghLastRequest.audios} 个。`)) : null,
        );
    }
    return {
        id: 'ghlink-video', name: 'GHLINK 视频与图片', version: '1.3.3', description: '站内大图预览和文件下载、清晰的任务按钮，支持 @ 参考素材及节点重建后继续等待。',
        css: `
            .ghlink-actions { display: flex; flex-wrap: wrap; align-items: center; gap: 8px; margin: 4px 0; }
            button.ghlink-action { display: inline-flex; align-items: center; justify-content: center; box-sizing: border-box; min-height: 36px; padding: 8px 14px; border: 1px solid var(--gh-button-border); border-radius: 8px; background: var(--gh-button-panel); color: var(--gh-button-text); font-family: inherit; font-size: 12px; font-weight: 600; line-height: 1.4; white-space: nowrap; cursor: pointer; transition: background-color .15s, border-color .15s, box-shadow .15s; }
            button.ghlink-action--primary { background: #2563eb; border-color: #2563eb; color: #fff; box-shadow: 0 2px 5px #2563eb26; }
            button.ghlink-action--secondary:hover:not(:disabled) { border-color: #3b82f6; box-shadow: 0 1px 4px #2563eb1a; }
            button.ghlink-action--primary:hover:not(:disabled) { background: #1d4ed8; border-color: #1d4ed8; }
            button.ghlink-action:focus-visible { outline: 2px solid #3b82f6; outline-offset: 2px; }
            button.ghlink-action:disabled { background: var(--gh-button-panel); color: var(--gh-button-muted); border-color: var(--gh-button-border); box-shadow: none; opacity: .6; cursor: not-allowed; }
            .ghlink-result { height: 100%; box-sizing: border-box; display: flex; flex-direction: column; gap: 8px; padding: 12px; overflow: auto; color: var(--gh-button-text); }
            .ghlink-result-media { position: relative; flex: 1; min-height: 80px; overflow: hidden; border: 1px solid var(--gh-button-border); border-radius: 8px; }
            .ghlink-image-preview { display: block; position: relative; width: 100%; height: 100%; padding: 0; border: 0; background: transparent; color: var(--gh-button-text); cursor: zoom-in; }
            .ghlink-image-preview img { display: block; width: 100%; height: 100%; object-fit: contain; }
            .ghlink-image-preview:focus-visible { outline: 2px solid #3b82f6; outline-offset: -3px; }
            .ghlink-image-hint { position: absolute; bottom: 6px; right: 6px; padding: 4px 7px; border: 1px solid var(--gh-button-border); border-radius: 4px; background: var(--gh-button-panel); color: var(--gh-button-text); font-size: 10px; }
            .ghlink-result-caption { display: flex; align-items: center; justify-content: space-between; gap: 8px; font-size: 12px; }
            .ghlink-media-meta { color: var(--gh-button-muted); font-size: 11px; line-height: 1.5; overflow-wrap: anywhere; }
            .ghlink-media-meta:empty { display: none; }
            .ghlink-media-empty { display: grid; height: 100%; place-content: center; gap: 10px; text-align: center; font-size: 12px; }
            dialog.ghlink-viewer { box-sizing: border-box; width: min(1100px, calc(100vw - 32px)); height: min(840px, calc(100dvh - 32px)); max-width: none; max-height: none; margin: auto; padding: 0; border: 1px solid var(--gh-button-border); border-radius: 12px; background: var(--gh-button-panel); color: var(--gh-button-text); box-shadow: 0 20px 70px #0005; overflow: hidden; }
            dialog.ghlink-viewer[open] { display: flex; flex-direction: column; }
            dialog.ghlink-viewer::backdrop { background: #0009; }
            .ghlink-viewer-header { display: grid; grid-template-columns: minmax(0, 1fr) auto; align-items: center; gap: 10px; padding: 14px 16px; border-bottom: 1px solid var(--gh-button-border); }
            .ghlink-viewer-tools { grid-column: 1 / -1; display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
            .ghlink-viewer-scale { min-width: 42px; text-align: center; font-size: 12px; font-variant-numeric: tabular-nums; }
            .ghlink-viewer-stage { flex: 1; min-height: 0; overflow: auto; overscroll-behavior: contain; }
            .ghlink-viewer-image-wrap { display: grid; place-items: center; min-width: 100%; min-height: 100%; }
            .ghlink-viewer-image-wrap img { display: block; object-fit: contain; }
            .ghlink-viewer-footer { padding: 8px 16px; border-top: 1px solid var(--gh-button-border); color: var(--gh-button-muted); font-size: 12px; }
        `,
        setup: () => () => {
            for (const session of sessions.values()) { session.controller.current?.abort(); session.listeners.clear(); }
            sessions.clear();
            syncReloadGuard();
            for (const url of resultURLs.values()) URL.revokeObjectURL(url); resultURLs.clear();
            for (const [url, timer] of downloadURLs) { clearTimeout(timer); URL.revokeObjectURL(url); } downloadURLs.clear();
        },
        nodes: [{ type: 'ghlink-video:generator', title: 'GHLINK 视频 / 图片', icon: '🎬', defaultSize: { width: 440, height: 760 }, hasSourceHandle: true, hidePanel: true, Content,
            resource: node => {
                const result = node.metadata?.ghResultURL;
                const url = safeURL(result) || (node.metadata?.ghMode === 'image' && /^data:image\/(png|jpeg|webp);base64,/.test(result || '') ? result : '') || resultURLs.get(node.metadata?.ghResultNodeId);
                return url ? { kind: node.metadata.ghMode || 'video', url } : null;
            } }, { type: 'ghlink-video:result', title: 'GHLINK 生成结果', icon: '🎞', defaultSize: { width: 360, height: 300 }, showInCreateMenu: false, hasSourceHandle: true, hidePanel: true, Content: Result, resource: node => { const url = node.metadata?.ghResultURL || resultURLs.get(node.id); return url ? { kind: node.metadata?.ghMode || 'video', url } : null; } }],
    };
}
