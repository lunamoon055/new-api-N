import { useEffect, useMemo, useState } from 'react'
import { BookOpen, Check, ChevronDown, ChevronRight, Copy, FileCode2, FileText, Image, Info, Menu, MessageSquare, Search, ShieldCheck, Terminal, Video, X, Box, RefreshCw } from 'lucide-react'
import { getCodeSample, getModelDoc, getModelKind, getRequestParams, getResponseSample } from './model-data.js'
import { useModelCatalog } from './use-model-catalog.js'
import { getAllModelDocs } from './all-model-docs.js'

const categories = [
  { id: 'text', label: '文本模型', icon: MessageSquare },
  { id: 'image', label: '图片模型', icon: Image },
  { id: 'video', label: '视频模型', icon: Video },
  { id: 'other', label: '其他接口', icon: Box },
]
const languages = ['cURL', 'Python', 'TypeScript']
const appOrigin = import.meta.env.DEV ? 'http://localhost:3001' : (window.location.protocol === 'file:' ? '' : window.location.origin)

function SyntaxLine({ text }) {
  return text.split(/("(?:[^"\\]|\\.)*"|'[^']*'|\b(?:curl|POST|GET|import|from|const|await|if|throw|new)\b|\b\d+\b)/g).map((token, index) => {
    let color = ''
    if (/^["']/.test(token)) color = 'syntax-string'
    else if (/^\d+$/.test(token)) color = 'syntax-number'
    else if (/^(curl|POST|GET|import|from|const|await|if|throw|new)$/.test(token)) color = 'syntax-keyword'
    return <span className={color} key={index}>{token}</span>
  })
}
function Code({ value }) {
  return <div className='code-scroll'><pre>{value.split('\n').map((line, index) => <div className='code-line' key={index}><span className='line-number' aria-hidden='true'>{index + 1}</span><code><SyntaxLine text={line} /></code></div>)}</pre></div>
}

export function App() {
  const { catalog, selectedName, setSelectedName, loading, error, updatedAt, loadCatalog } = useModelCatalog()
  const [language, setLanguage] = useState('cURL')
  const [search, setSearch] = useState('')
  const [searchOpen, setSearchOpen] = useState(false)
  const [menuOpen, setMenuOpen] = useState(false)
  const [closedGroups, setClosedGroups] = useState(new Set())
  const [toast, setToast] = useState('')
  const [manualCopy, setManualCopy] = useState('')
  useEffect(() => {
    if (!toast) return
    const timer = window.setTimeout(() => setToast(''), 2300)
    return () => window.clearTimeout(timer)
  }, [toast])
  useEffect(() => {
    function onKey(event) {
      if (event.key === 'Escape') { setMenuOpen(false); setSearchOpen(false); setManualCopy('') }
      if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
        event.preventDefault()
        document.getElementById('docs-search')?.focus()
      }
    }
    window.addEventListener('keydown', onKey)
    return () => window.removeEventListener('keydown', onKey)
  }, [])

  const selectedModel = catalog.models.find((model) => model.model_name === selectedName)
  const doc = selectedModel ? getModelDoc(selectedModel, catalog.endpoints) : null
  const code = selectedModel && doc ? getCodeSample(selectedModel, doc, language) : ''
  const responseCode = doc ? getResponseSample(doc) : ''
  const params = doc ? getRequestParams(doc) : []
  const filteredModels = useMemo(() => catalog.models.filter((model) => model.model_name.toLowerCase().includes(search.trim().toLowerCase())), [catalog.models, search])
  const categoryLabel = categories.find((category) => category.id === doc?.kind)?.label || '模型目录'
  function selectModel(name) {
    setSelectedName(name); setSearch(''); setSearchOpen(false); setMenuOpen(false)
    document.getElementById('guide')?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  function toggleGroup(id) {
    setClosedGroups((current) => {
      const next = new Set(current)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }
  async function copy(value, label = '代码') {
    try { await navigator.clipboard.writeText(value); setToast(label + '已复制') }
    catch { setManualCopy(value) }
  }
  function copyAllModels() {
    if (loading || error || !catalog.models.length) return
    const baseUrl = import.meta.env.DEV ? 'http://localhost:3000' : window.location.origin
    copy(getAllModelDocs(catalog, baseUrl), `${catalog.models.length} 个模型的接入说明`)
  }
  function closeNavigation() { setMenuOpen(false) }

  return <>
    <header className='site-header'>
      <a className='brand' href='#guide' aria-label='New API 模型接入指南'><img src='/brand-logo.png' alt='' /><span>New API</span></a>
      <nav className='top-nav' aria-label='主导航'><a href={appOrigin || '#guide'} target={appOrigin ? '_blank' : undefined} rel='noreferrer'>产品</a><a href={appOrigin ? appOrigin + '/pricing' : '#models'} target={appOrigin ? '_blank' : undefined} rel='noreferrer'>价格</a><a className='active' href='#guide'>文档</a><a href={appOrigin ? appOrigin + '/dashboard' : '#auth'} target={appOrigin ? '_blank' : undefined} rel='noreferrer'>控制台</a></nav>
      <div className='header-tools'>
        <div className='search-wrap'><Search size={17} aria-hidden='true' /><input id='docs-search' type='search' value={search} onChange={(event) => { setSearch(event.target.value); setSearchOpen(true) }} onFocus={() => setSearchOpen(true)} onBlur={() => window.setTimeout(() => setSearchOpen(false), 150)} placeholder='搜索模型文档…' aria-label='搜索模型文档' /><kbd>⌘ K</kbd>
          {searchOpen && search.trim() && <div className='search-results'><p>{filteredModels.length} 个匹配模型</p>{filteredModels.length ? filteredModels.slice(0, 7).map((model) => <button key={model.model_name} onMouseDown={(event) => event.preventDefault()} onClick={() => selectModel(model.model_name)}><FileCode2 size={16} /><span>{model.model_name}</span><ChevronRight size={15} /></button>) : <div className='search-empty'>没有找到模型，试试其他关键词。</div>}</div>}
        </div>
        <a className='help-link' href='#faq'>帮助</a><button className='mobile-menu icon-button' aria-label={menuOpen ? '关闭模型目录' : '打开模型目录'} aria-expanded={menuOpen} onClick={() => setMenuOpen(!menuOpen)}>{menuOpen ? <X /> : <Menu />}</button>
      </div>
    </header>
    {menuOpen && <button className='menu-backdrop' aria-label='关闭菜单' onClick={() => setMenuOpen(false)} />}
    <aside className={'sidebar ' + (menuOpen ? 'is-open' : '')} aria-label='文档导航'><nav>
      <div className='nav-section'><h2>接入文档</h2><a className='nav-link selected' href='#guide' onClick={closeNavigation}><FileText size={18} />模型接入指南</a><a className='nav-link' href='#endpoint' onClick={closeNavigation}><FileCode2 size={18} />请求接口</a><a className='nav-link' href='#faq' onClick={closeNavigation}><BookOpen size={18} />常见问题</a></div>
      <div className='nav-section'><h2>鉴权</h2><a className='nav-link' href='#auth' onClick={closeNavigation}><ShieldCheck size={18} />请求鉴权</a><a className='nav-link' href='#security' onClick={closeNavigation}><ShieldCheck size={18} />安全最佳实践</a></div>
      <div className='nav-section model-directory' id='models'><div className='directory-title'><h2>模型目录</h2><span>{catalog.models.length}</span><button className='icon-button refresh' onClick={loadCatalog} aria-label='刷新模型目录' disabled={loading}><RefreshCw size={14} className={loading ? 'spin' : ''} /></button></div>
        {loading && <div className='directory-loading' role='status' aria-label='正在读取模型列表'><span /><span /><span /></div>}{error && <div className='directory-error'><p>{error}</p></div>}
        {!loading && categories.map((category) => {
          const models = filteredModels.filter((model) => getModelKind(model) === category.id)
          if (!models.length) return null
          const Icon = category.icon
          const open = !closedGroups.has(category.id)
          return <div className='model-group' key={category.id}><button className='group-toggle' aria-expanded={open} onClick={() => toggleGroup(category.id)}><Icon size={18} /><span>{category.label}</span><span className='group-count'>{models.length}</span><ChevronDown size={14} className={open ? 'chevron open' : 'chevron'} /></button>{open && <div className='model-links'>{models.map((model) => <button key={model.model_name} className={model.model_name === selectedName ? 'model-link active' : 'model-link'} onClick={() => selectModel(model.model_name)} title={model.model_name} aria-current={model.model_name === selectedName ? 'page' : undefined}>{model.model_name}</button>)}</div>}</div>
        })}
        {!loading && !error && !filteredModels.length && <p className='directory-state'>{search ? '没有匹配的模型' : '网站模型列表为空'}</p>}
      </div>
    </nav><div className='sidebar-footer'>{updatedAt ? `已同步 ${updatedAt.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}，每分钟自动刷新` : '目录来自网站实时模型列表'}</div></aside>

    <main className='workspace' id='guide'><div className='article-and-code'>
      <article className='article'>
        <nav className='breadcrumb' aria-label='面包屑'><a href='#guide'>文档</a><ChevronRight size={13} /><a href='#models'>模型目录</a><ChevronRight size={13} /><span>{categoryLabel}</span><ChevronRight size={13} /><strong>{selectedName || '加载中'}</strong></nav>
        <div className='page-heading'><h1>模型接入指南</h1><button className='copy-all-button' onClick={copyAllModels} disabled={loading || !!error || !catalog.models.length}><Copy size={15} />复制全部模型对接</button></div><p className='page-subtitle'>{selectedName ? `查看 ${selectedName} 的接入接口、参数和调用示例。` : '根据网站当前模型列表展示接入接口、参数和调用示例。'}</p>
        <div className='notice'><Info size={20} /><p>模型可用性与令牌权限有关，请通过 <code>GET</code> <code>/v1/models</code> 获取当前可用模型。</p></div>
        {selectedModel && doc ? <>
          <section className='about-model'><div className='section-heading'><h2>关于 {selectedName}</h2><button className='small-copy' onClick={() => copy(selectedName, '模型 ID')} aria-label='复制完整模型 ID'><Copy size={14} /></button></div><p>{selectedModel.description || (doc.kind === 'image' ? '通过图片生成接口，将文字描述转换为图像。请使用完整模型 ID，并根据该模型的能力设置画面比例和分辨率。' : doc.kind === 'video' ? '通过视频任务接口提交生成请求。创建成功后保存任务 ID，再查询处理进度与最终结果。' : '根据站点公开的接口信息接入此模型。模型名称中的大小写、中文与符号都是 ID 的一部分。')}</p></section>
          <section id='endpoint' className='endpoint-section'><h2>请求接口</h2><div className='endpoint-row'><span className='http-method'>{doc.method}</span><code>{doc.verified ? doc.path : '端点信息待核对'}</code>{doc.verified && <button className='small-copy' onClick={() => copy(doc.path, '接口路径')} aria-label='复制接口路径'><Copy size={15} /></button>}</div><p>{doc.verified ? (doc.kind === 'video' ? '创建请求仅代表任务已受理，需继续查询结果。' : '请求头需要携带 API 令牌进行鉴权。') : '当前模型缺少明确的专用端点文档，不自动生成调用示例。请核对模型详情与管理员配置。'}</p><span className='source-note'>信息来源：{doc.source}</span>{!doc.verified && doc.path && <div className='unverified-note'>站点目前声明 <code>{doc.path}</code>。此信息需要结合模型用途进一步确认。</div>}{doc.queryPath && <div className='query-endpoint'><span>查询任务</span><code>GET {doc.queryPath}</code><button className='small-copy' onClick={() => copy('curl "$API_BASE_URL' + doc.queryPath.replace('{task_id}', '$TASK_ID') + '" -H "Authorization: Bearer $API_KEY"', '查询示例')} aria-label='复制查询任务示例'><Copy size={14} /></button></div>}</section>
          {params.length > 0 && <section className='params-section'><h3>请求参数</h3><div className='table-scroll'><table><thead><tr><th>参数</th><th>类型</th><th>是否必填</th><th>说明</th></tr></thead><tbody>{params.map((param) => <tr key={param[0]}><td><code>{param[0]}</code></td><td>{param[1]}</td><td>{param[2]}</td><td>{param[3]}</td></tr>)}</tbody></table></div></section>}
        </> : <section className='main-state'><Terminal size={32} /><h2>{loading ? '正在读取模型目录' : error ? '模型目录暂不可用' : '暂无可展示的模型'}</h2><p>{loading ? '正在同步网站当前模型列表。' : error || '在网站添加并启用模型后，接入文档会自动显示。'}</p>{!loading && <button className='primary-button' onClick={loadCatalog}>重新加载</button>}</section>}
        <section id='auth' className='lower-section'><h2>请求鉴权</h2><p>使用本网关签发的 API Key，在服务端请求中携带以下请求头：</p><div className='auth-header'><code>Authorization: Bearer &lt;API_KEY&gt;</code><button className='small-copy' onClick={() => copy('Authorization: Bearer <API_KEY>', '鉴权请求头')} aria-label='复制鉴权请求头'><Copy size={15} /></button></div><p>先在服务端设置环境变量，将示例域名和令牌替换为自己的配置：</p><div className='auth-header'><code>export API_BASE_URL="https://your-gateway.example.com"<br />export API_KEY="&lt;YOUR_API_KEY&gt;"</code><button className='small-copy' onClick={() => copy('export API_BASE_URL="https://your-gateway.example.com"\nexport API_KEY="<YOUR_API_KEY>"', '环境变量示例')} aria-label='复制环境变量示例'><Copy size={15} /></button></div><p>Anthropic 等协议可能使用其他请求头，请遵循所选端点的专用示例。</p></section>
        <section id='security' className='lower-section'><h2>安全最佳实践</h2><ul className='security-list'><li>密钥只保存在服务端环境变量中。</li><li>按模型、分组和 IP 设置最小必要权限。</li><li>日志与报错信息中隐藏完整 API Key。</li></ul></section>
        <section id='faq' className='lower-section'><h2>常见问题</h2><details><summary>为什么模型广场能看到模型，调用却返回 403？</summary><p>模型目录展示站点配置，实际权限取决于当前令牌和分组。请用该令牌调用 GET /v1/models，核对可用的完整模型 ID。</p></details><details><summary>视频请求返回 queued，是否已经生成成功？</summary><p>queued 表示排队中。保存 task_id，使用查询端点轮询，直到任务完成或失败。</p></details><details><summary>模型切换后，可以直接复用所有参数吗？</summary><p>不同模型的字段、时长、分辨率与素材限制可能不同。仅发送当前模型明确支持的参数。</p></details></section>
        <footer className='article-footer'>© 2026 <a href='https://github.com/QuantumNous/new-api' target='_blank' rel='noreferrer'>New API</a> · QuantumNous</footer>
      </article>
      <aside className='code-column' aria-label='调用示例'><div className='code-panel'>
        <div className='panel-heading'><h2>请求示例</h2><button className='dark-copy' onClick={() => copy(code)} disabled={!code}><Copy size={15} />复制代码</button></div>
        <div className='code-tabs' role='tablist' aria-label='示例语言'>{languages.map((item) => <button key={item} role='tab' aria-selected={language === item} className={language === item ? 'selected' : ''} onClick={() => setLanguage(item)}>{item}</button>)}</div>
        {code ? <Code value={code} /> : <div className='code-empty'><FileCode2 size={28} /><h3>{loading ? '正在准备示例' : selectedModel ? '需要确认接口信息' : '暂无调用示例'}</h3><p>{selectedModel ? '确认专用端点与参数后，才能生成适用于此模型的调用代码。' : '网站模型列表加载成功后，将展示对应调用示例。'}</p></div>}
        <button className='primary-button copy-request' onClick={() => copy(code, '请求示例')} disabled={!code}><Copy size={17} />复制请求示例</button><p className='env-note'>使用环境变量 <code>API_BASE_URL</code> 与 <code>API_KEY</code></p>
        <div className='response-section'><div className='panel-heading'><h2>返回示例</h2><button className='dark-copy' onClick={() => copy(responseCode, '返回示例')} disabled={!responseCode}><Copy size={15} />复制代码</button></div>{responseCode ? <Code value={responseCode} /> : <p className='response-unavailable'>确认接口后展示响应结构。</p>}<p className='response-caption'>响应结构示意，实际内容以接口返回为准。</p></div>
      </div><p className='code-footnote'><ShieldCheck size={14} />API Key 只在你的服务端使用</p></aside>
    </div></main>
    {toast && <div className='toast' role='status'><Check size={17} />{toast}</div>}
    {manualCopy && <div className='copy-dialog-backdrop'><section className='copy-dialog' role='dialog' aria-modal='true' aria-labelledby='manual-copy-title'><div className='section-heading'><h2 id='manual-copy-title'>手动复制</h2><button className='icon-button' aria-label='关闭手动复制' onClick={() => setManualCopy('')}><X size={20} /></button></div><p>浏览器未允许写入剪贴板。下方内容已选中，按 Ctrl+C 或 ⌘C 复制。</p><label htmlFor='manual-copy-content'>接入说明</label><textarea id='manual-copy-content' readOnly value={manualCopy} autoFocus onFocus={(event) => event.target.select()} /></section></div>}
  </>
}
