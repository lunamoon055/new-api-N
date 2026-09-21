/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { useMemo, useState } from 'react'
import {
  ChevronRight,
  ExternalLink,
  Gauge,
  KeyRound,
  ScrollText,
  ShieldCheck,
  Sigma,
  Zap,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import type { BundledLanguage } from 'shiki/bundle/web'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { Badge } from '@/components/ui/badge'
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table'
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs'
import {
  CodeBlock,
  CodeBlockCopyButton,
} from '@/components/ai-elements/code-block'
import {
  buildRateLimits,
  buildSupportedParameters,
  formatRateLimit,
  type SupportedParameter,
} from '../lib/mock-stats'
import { replaceModelInPath } from '../lib/model-helpers'
import { inferApiInfo } from '../lib/model-metadata'
import { getModelEndpointPreset } from '../lib/model-endpoint-presets'
import type { PricingEndpoint, PricingModel } from '../types'

// ---------------------------------------------------------------------------
// Code-sample registry
// ---------------------------------------------------------------------------
//
// Each sample is keyed by language and endpoint type. The endpoint type comes
// from the model's `supported_endpoint_types`; we render samples only for the
// types the model actually supports. This keeps copy-pasted code accurate and
// provider-shaped (OpenAI vs Anthropic vs Gemini, etc.).

type Lang = 'curl' | 'python' | 'typescript' | 'javascript'

const LANG_LABELS: Record<Lang, string> = {
  curl: 'cURL',
  python: 'Python',
  typescript: 'TypeScript',
  javascript: 'JavaScript',
}

const LANG_HIGHLIGHT: Record<Lang, BundledLanguage> = {
  curl: 'bash',
  python: 'python',
  typescript: 'typescript',
  javascript: 'javascript',
}

type SampleContext = {
  baseUrl: string
  apiKeyEnv: string
  modelName: string
  endpointType: string
  endpointPath: string
}

type DisplayEndpoint = {
  type: string
  labelKey: string
  kind?: 'video' | 'image'
  isPreset?: boolean
  path: string
  method: string
  queryPath?: string
  queryMethod?: string
}

function buildChatSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const isResponses = ctx.endpointType === 'openai-response'
  const isReasoning = /^o[1-4]|reasoning|thinking|deepseek-r/i.test(
    ctx.modelName
  )
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  const bodyJson = isResponses
    ? JSON.stringify({ model: ctx.modelName, input: userMessage }, null, 2)
    : JSON.stringify(
        {
          model: ctx.modelName,
          messages: [{ role: 'user', content: userMessage }],
          ...(isReasoning ? {} : { temperature: 0.7 }),
        },
        null,
        2
      )

  const fnCall = isResponses ? 'responses.create' : 'chat.completions.create'

  if (lang === 'curl') {
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${bodyJson.replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      'client = OpenAI(',
      `    base_url="${ctx.baseUrl}/v1",`,
      `    api_key="<YOUR_API_KEY>",`,
      ')',
      '',
      isResponses
        ? `response = client.${fnCall}(\n    model="${ctx.modelName}",\n    input="${userMessage}",\n)\n\nprint(response.output_text)`
        : `completion = client.${fnCall}(\n    model="${ctx.modelName}",\n    messages=[\n        {"role": "user", "content": "${userMessage}"}\n    ],\n)\n\nprint(completion.choices[0].message.content)`,
    ].join('\n')
  }

  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      isResponses
        ? `const response = await client.${fnCall}({\n  model: '${ctx.modelName}',\n  input: '${userMessage}',\n})\n\nconsole.log(response.output_text)`
        : `const completion = await client.${fnCall}({\n  model: '${ctx.modelName}',\n  messages: [{ role: 'user', content: '${userMessage}' }],\n})\n\nconsole.log(completion.choices[0].message.content)`,
    ].join('\n')
  }

  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyJson}),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data)`,
  ].join('\n')
}

function buildAnthropicSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      {
        model: ctx.modelName,
        max_tokens: 1024,
        messages: [{ role: 'user', content: userMessage }],
      },
      null,
      2
    )
    return [
      `curl ${url} \\`,
      `  -H "x-api-key: $${ctx.apiKeyEnv}" \\`,
      `  -H "anthropic-version: 2023-06-01" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import anthropic',
      '',
      'client = anthropic.Anthropic(',
      `    base_url="${ctx.baseUrl}",`,
      `    api_key="<YOUR_API_KEY>",`,
      ')',
      '',
      `message = client.messages.create(`,
      `    model="${ctx.modelName}",`,
      `    max_tokens=1024,`,
      `    messages=[{"role": "user", "content": "${userMessage}"}],`,
      ')',
      '',
      'print(message.content[0].text)',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import Anthropic from '@anthropic-ai/sdk'`,
      '',
      `const client = new Anthropic({`,
      `  baseURL: '${ctx.baseUrl}',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const message = await client.messages.create({`,
      `  model: '${ctx.modelName}',`,
      `  max_tokens: 1024,`,
      `  messages: [{ role: 'user', content: '${userMessage}' }],`,
      `})`,
      '',
      `console.log(message.content[0].text)`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    'x-api-key': process.env.${ctx.apiKeyEnv},`,
    `    'anthropic-version': '2023-06-01',`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    max_tokens: 1024,`,
    `    messages: [{ role: 'user', content: '${userMessage}' }],`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.content[0].text)`,
  ].join('\n')
}

function buildGeminiSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}?key=$${ctx.apiKeyEnv}`
  const userMessage = 'Explain quantum entanglement in one paragraph.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      { contents: [{ parts: [{ text: userMessage }] }] },
      null,
      2
    )
    return [
      `curl '${url}' \\`,
      `  -H 'Content-Type: application/json' \\`,
      `  -d '${body.replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'import google.generativeai as genai',
      '',
      `genai.configure(api_key="<YOUR_API_KEY>")`,
      '',
      `model = genai.GenerativeModel("${ctx.modelName}")`,
      `response = model.generate_content("${userMessage}")`,
      '',
      `print(response.text)`,
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import { GoogleGenerativeAI } from '@google/generative-ai'`,
      '',
      `const genAI = new GoogleGenerativeAI(process.env.${ctx.apiKeyEnv}!)`,
      `const model = genAI.getGenerativeModel({ model: '${ctx.modelName}' })`,
      '',
      `const result = await model.generateContent('${userMessage}')`,
      `console.log(result.response.text())`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: { 'Content-Type': 'application/json' },`,
    `  body: JSON.stringify({`,
    `    contents: [{ parts: [{ text: '${userMessage}' }] }],`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.candidates[0].content.parts[0].text)`,
  ].join('\n')
}

function buildEmbeddingSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const text = 'The food was delicious and the waiter…'

  if (lang === 'curl') {
    const body = JSON.stringify({ model: ctx.modelName, input: text }, null, 2)
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.embeddings.create(',
      `    model="${ctx.modelName}",`,
      `    input="${text}",`,
      ')',
      '',
      'print(response.data[0].embedding[:8])',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const response = await client.embeddings.create({`,
      `  model: '${ctx.modelName}',`,
      `  input: '${text}',`,
      `})`,
      '',
      `console.log(response.data[0].embedding.slice(0, 8))`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    input: '${text}',`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.data[0].embedding.slice(0, 8))`,
  ].join('\n')
}

function buildImageSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = 'A serene koi pond at sunset, ukiyo-e style.'

  if (lang === 'curl') {
    const body = JSON.stringify(
      { model: ctx.modelName, prompt, size: '1024x1024', n: 1 },
      null,
      2
    )
    return [
      `curl ${url} \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${body.replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }
  if (lang === 'python') {
    return [
      'from openai import OpenAI',
      '',
      `client = OpenAI(base_url="${ctx.baseUrl}/v1", api_key="<YOUR_API_KEY>")`,
      '',
      'response = client.images.generate(',
      `    model="${ctx.modelName}",`,
      `    prompt="${prompt}",`,
      `    size="1024x1024",`,
      `    n=1,`,
      ')',
      '',
      'print(response.data[0].url)',
    ].join('\n')
  }
  if (lang === 'typescript') {
    return [
      `import OpenAI from 'openai'`,
      '',
      `const client = new OpenAI({`,
      `  baseURL: '${ctx.baseUrl}/v1',`,
      `  apiKey: process.env.${ctx.apiKeyEnv},`,
      `})`,
      '',
      `const response = await client.images.generate({`,
      `  model: '${ctx.modelName}',`,
      `  prompt: '${prompt}',`,
      `  size: '1024x1024',`,
      `  n: 1,`,
      `})`,
      '',
      `console.log(response.data[0].url)`,
    ].join('\n')
  }
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify({`,
    `    model: '${ctx.modelName}',`,
    `    prompt: '${prompt}',`,
    `    size: '1024x1024',`,
    `    n: 1,`,
    `  }),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.data[0].url)`,
  ].join('\n')
}

function buildImageAsyncSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const normalizedModel = ctx.modelName.trim().toLowerCase()
  const outputResolution = normalizedModel.startsWith('nano-banana')
    ? '1K'
    : '2K'
  const body = {
    model: ctx.modelName,
    prompt: 'A minimal poster of a sunset over the sea.',
    aspect_ratio: '1:1',
    output_resolution: outputResolution,
  }

  if (lang === 'curl') {
    return [
      `curl -X POST "${url}" \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${JSON.stringify(body, null, 2).replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    return [
      'import requests',
      '',
      'response = requests.post(',
      `    "${url}",`,
      `    headers={"Authorization": "Bearer <YOUR_API_KEY>"},`,
      `    json=${JSON.stringify(body, null, 2)},`,
      ')',
      'task = response.json()',
      'print(task["task_id"])',
    ].join('\n')
  }

  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${JSON.stringify(body, null, 2)}),`,
    `})`,
    '',
    `const task = await response.json()`,
    `console.log(task.task_id)`,
  ].join('\n')
}

function buildVideoSample(lang: Lang, ctx: SampleContext): string {
  const url = `${ctx.baseUrl}${ctx.endpointPath}`
  const prompt = 'A cinematic shot of a cat walking through a sunlit garden.'

  // 根据模型名称智能选择参数
  const modelLower = ctx.modelName.toLowerCase()
  const isVideosApi =
    modelLower.startsWith('videos-') ||
    modelLower.startsWith('sd2') ||
    modelLower.startsWith('004系列/')

  // 构建请求体
  const body: Record<string, any> = {
    model: ctx.modelName,
    prompt,
  }

  // 为 videos API 格式添加特定参数
  if (isVideosApi) {
    body.ratio = '16:9'
    body.resolution = '720p'
    body.duration = 5
  } else {
    // 为异步视频 API 添加参数
    body.size = '1280x720'
    body.duration = 4
  }

  if (lang === 'curl') {
    return [
      `curl -X POST "${url}" \\`,
      `  -H "Authorization: Bearer $${ctx.apiKeyEnv}" \\`,
      `  -H "Content-Type: application/json" \\`,
      `  -d '${JSON.stringify(body, null, 2).replace(/\n/g, '\n     ')}'`,
    ].join('\n')
  }

  if (lang === 'python') {
    const pythonBody = Object.entries(body)
      .map(([key, value]) => `    "${key}": ${JSON.stringify(value)},`)
      .join('\n')
    return [
      'import requests',
      '',
      'response = requests.post(',
      `    "${url}",`,
      `    headers={"Authorization": "Bearer <YOUR_API_KEY>"},`,
      `    json={`,
      pythonBody,
      '    }',
      ')',
      'print(response.json())',
    ].join('\n')
  }

  const bodyText = JSON.stringify(body, null, 2)
  return [
    `const response = await fetch('${url}', {`,
    `  method: 'POST',`,
    `  headers: {`,
    `    Authorization: \`Bearer \${process.env.${ctx.apiKeyEnv}}\`,`,
    `    'Content-Type': 'application/json',`,
    `  },`,
    `  body: JSON.stringify(${bodyText}),`,
    `})`,
    '',
    `const data = await response.json()`,
    `console.log(data.task_id) // 使用 task_id 查询任务状态`,
  ].join('\n')
}

function buildSample(
  lang: Lang,
  endpointType: string,
  ctx: SampleContext
): string {
  if (endpointType === 'anthropic') return buildAnthropicSample(lang, ctx)
  if (endpointType === 'gemini') return buildGeminiSample(lang, ctx)
  if (endpointType === 'embeddings' || endpointType === 'jina-rerank')
    return buildEmbeddingSample(lang, ctx)
  if (endpointType === 'image-generation') return buildImageSample(lang, ctx)
  if (
    endpointType === 'openai-video' ||
    endpointType === 'video-async' ||
    endpointType === 'video-generations' ||
    endpointType === 'video-videos'
  )
    return buildVideoSample(lang, ctx)
  if (endpointType === 'image-async') return buildImageAsyncSample(lang, ctx)
  return buildChatSample(lang, ctx)
}

// ---------------------------------------------------------------------------
// Code samples section
// ---------------------------------------------------------------------------

function CodeSamplesSection(props: {
  model: PricingModel
  endpointMap: Record<string, PricingEndpoint>
}) {
  const { t } = useTranslation()
  const { status } = useStatus()

  const baseUrl = useMemo(() => {
    const candidate =
      (status as Record<string, unknown> | null)?.server_address ??
      (status as Record<string, unknown> | null)?.serverAddress ??
      (status?.data as Record<string, unknown> | undefined)?.server_address ??
      (status?.data as Record<string, unknown> | undefined)?.serverAddress
    if (candidate && typeof candidate === 'string') {
      return candidate.replace(/\/$/, '')
    }
    if (typeof window !== 'undefined') return window.location.origin
    return ''
  }, [status])

  const endpoints = useMemo(() => {
    const types = props.model.supported_endpoint_types || []
    const preset = getModelEndpointPreset(props.model.model_name)
    const configured: DisplayEndpoint[] = types
      .map((type) => {
        const info = props.endpointMap[type] || {}
        let path = info.path || ''
        if (path && path.includes('{model}')) {
          path = replaceModelInPath(path, props.model.model_name || '')
        }
        let queryPath = info.query_path || ''
        if (queryPath && queryPath.includes('{model}')) {
          queryPath = replaceModelInPath(
            queryPath,
            props.model.model_name || ''
          )
        }
        return {
          type,
          labelKey: '',
          path,
          method: info.method || 'POST',
          ...(queryPath ? { queryPath } : {}),
          ...(info.query_method
            ? { queryMethod: info.query_method }
            : queryPath
              ? { queryMethod: 'GET' }
              : {}),
        }
      })
      .filter((e) => Boolean(e.path))

    const presetEndpoint: DisplayEndpoint | null = preset
      ? {
          type: preset.type,
          labelKey: preset.labelKey,
          kind: preset.kind,
          isPreset: !configured.some(
            (endpoint) =>
              endpoint.path === preset.path &&
              endpoint.method === preset.method &&
              endpoint.queryPath === preset.query_path
          ),
          path: preset.path || '',
          method: preset.method || 'POST',
          queryPath: preset.query_path,
          queryMethod: preset.query_method,
        }
      : null
    const candidates: DisplayEndpoint[] = presetEndpoint
      ? [presetEndpoint, ...configured]
      : configured
    const seen = new Set<string>()
    return candidates.filter((endpoint) => {
      // A documented video preset replaces generic Chat/video metadata for
      // that model, while preserving other explicitly configured media APIs.
      if (preset && preset.kind === 'video' && endpoint.type === 'openai') {
        return false
      }
      if (
        preset &&
        preset.kind === 'video' &&
        endpoint.type === 'openai-video'
      ) {
        return false
      }
      if (
        preset &&
        preset.kind === 'image' &&
        endpoint.type === 'image-generation'
      ) {
        return false
      }
      const key = `${endpoint.method}|${endpoint.path}|${endpoint.queryMethod}|${endpoint.queryPath}`
      if (seen.has(key)) return false
      seen.add(key)
      return true
    })
  }, [props.model, props.endpointMap])

  const [endpointType, setEndpointType] = useState<string>(
    endpoints[0]?.type ?? ''
  )
  const [lang, setLang] = useState<Lang>('curl')

  const activeEndpoint = useMemo(() => {
    return endpoints.find((e) => e.type === endpointType) ?? endpoints[0]
  }, [endpointType, endpoints])

  if (endpoints.length === 0 || !activeEndpoint) {
    return null
  }

  const code = buildSample(lang, activeEndpoint.type, {
    baseUrl,
    apiKeyEnv: 'NEW_API_KEY',
    modelName: props.model.model_name || '',
    endpointType: activeEndpoint.type,
    endpointPath: activeEndpoint.path,
  })

  return (
    <section>
      <SectionTitle icon={ScrollText}>{t('Code samples')}</SectionTitle>

      <div className='flex flex-wrap items-center gap-2'>
        {endpoints.length > 1 && (
          <Tabs value={activeEndpoint.type} onValueChange={setEndpointType}>
            <TabsList className='bg-muted/40 h-8 p-0.5'>
              {endpoints.map((ep) => (
                <TabsTrigger
                  key={ep.type}
                  value={ep.type}
                  className='h-7 px-2.5 text-xs'
                >
                  {ep.labelKey ? t(ep.labelKey) : ep.type}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        )}

        <Tabs
          value={lang}
          onValueChange={(v) => setLang(v as Lang)}
          className='ml-auto'
        >
          <TabsList className='bg-muted/40 h-8 p-0.5'>
            {(Object.keys(LANG_LABELS) as Lang[]).map((l) => (
              <TabsTrigger key={l} value={l} className='h-7 px-2.5 text-xs'>
                {LANG_LABELS[l]}
              </TabsTrigger>
            ))}
          </TabsList>
        </Tabs>
      </div>

      <div className='border-border/60 bg-muted/20 mb-3 divide-y overflow-hidden rounded-lg border text-xs'>
        <div className='flex flex-wrap items-center gap-x-2 gap-y-1 p-3'>
          <span className='text-muted-foreground'>{t('Request endpoint')}</span>
          <Badge variant='outline' className='font-mono text-[10px]'>
            {activeEndpoint.method}
          </Badge>
          <code className='break-all font-mono text-[11px]'>
            {baseUrl}
            {activeEndpoint.path}
          </code>
        </div>
        {activeEndpoint.queryPath && (
          <div className='space-y-2 p-3'>
            <div className='flex flex-wrap items-center gap-x-2 gap-y-1'>
              <span className='text-muted-foreground'>
                {t('Query endpoint')}
              </span>
              <Badge variant='outline' className='font-mono text-[10px]'>
                {activeEndpoint.queryMethod}
              </Badge>
              <code className='break-all font-mono text-[11px]'>
                {baseUrl}
                {activeEndpoint.queryPath}
              </code>
            </div>
            <CodeBlock
              code={[
                `curl -X ${activeEndpoint.queryMethod} "${baseUrl}${activeEndpoint.queryPath.replace(/\{task_id\}/g, '$TASK_ID')}" \\`,
                `  -H "Authorization: Bearer $NEW_API_KEY"`,
              ].join('\n')}
              language='bash'
            >
              <CodeBlockCopyButton />
            </CodeBlock>
            <p className='text-muted-foreground text-[11px]'>
              {t(
                'Use the task_id returned by the creation request to query its status.'
              )}
            </p>
          </div>
        )}
      </div>
      {activeEndpoint.isPreset && (
        <p className='text-muted-foreground -mt-1 mb-3 text-[11px]'>
          {t(
            'This endpoint is preconfigured from the supplied channel documentation; verify model availability with GET /v1/models.'
          )}
        </p>
      )}

      <div className='mt-3'>
        <CodeBlock code={code} language={LANG_HIGHLIGHT[lang]}>
          <CodeBlockCopyButton />
        </CodeBlock>
      </div>

      <p className='text-muted-foreground mt-2 text-xs'>
        {t('Replace')}{' '}
        <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
          {'<YOUR_API_KEY>'}
        </code>{' '}
        {t('with the API key from your token settings.')}
      </p>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Supported parameters table
// ---------------------------------------------------------------------------

function SupportedParametersSection(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const params = useMemo(
    () => buildSupportedParameters(props.model),
    [props.model]
  )

  if (params.length === 0) return null

  return (
    <section>
      <SectionTitle icon={Sigma}>{t('Supported parameters')}</SectionTitle>
      <div className='border-border/60 overflow-hidden rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow className='bg-muted/30 hover:bg-muted/30'>
              <TableHead className='h-9 w-44 text-xs'>
                {t('Parameter')}
              </TableHead>
              <TableHead className='h-9 w-24 text-xs'>{t('Type')}</TableHead>
              <TableHead className='h-9 w-32 text-xs'>
                {t('Default / range')}
              </TableHead>
              <TableHead className='h-9 text-xs'>{t('Description')}</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {params.map((p) => (
              <TableRow key={p.name} className='hover:bg-muted/20'>
                <TableCell className='py-2 align-top'>
                  <div className='flex items-center gap-1.5'>
                    <code className='font-mono text-xs font-medium'>
                      {p.name}
                    </code>
                    {p.required && (
                      <Badge
                        variant='outline'
                        className='h-4 border-rose-500/40 px-1 text-[9px] text-rose-600 dark:text-rose-400'
                      >
                        {t('required')}
                      </Badge>
                    )}
                  </div>
                </TableCell>
                <TableCell className='py-2 align-top'>
                  <Badge
                    variant='secondary'
                    className='h-5 rounded-sm px-1.5 font-mono text-[10px] font-normal'
                  >
                    {p.type}
                  </Badge>
                </TableCell>
                <TableCell className='py-2 align-top'>
                  <ParamRangeCell param={p} />
                </TableCell>
                <TableCell className='text-muted-foreground py-2 align-top text-xs'>
                  {t(p.descriptionKey)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
    </section>
  )
}

function ParamRangeCell(props: { param: SupportedParameter }) {
  const { defaultValue, range, enumValues } = props.param
  if (defaultValue !== undefined) {
    return (
      <div className='flex flex-wrap items-center gap-1'>
        <span className='text-muted-foreground text-[11px]'>=</span>
        <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
          {String(defaultValue)}
        </code>
        {range && (
          <span className='text-muted-foreground text-[11px]'>{range}</span>
        )}
      </div>
    )
  }
  if (range) {
    return (
      <span className='text-muted-foreground font-mono text-[11px]'>
        {range}
      </span>
    )
  }
  if (enumValues && enumValues.length > 0) {
    return (
      <div className='flex flex-wrap gap-0.5'>
        {enumValues.map((v) => (
          <code
            key={v}
            className='bg-muted text-muted-foreground rounded px-1 py-0.5 font-mono text-[10px]'
          >
            {v}
          </code>
        ))}
      </div>
    )
  }
  return <span className='text-muted-foreground/60 text-[11px]'>—</span>
}

// ---------------------------------------------------------------------------
// Rate-limits table
// ---------------------------------------------------------------------------

function RateLimitsSection(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const limits = useMemo(() => buildRateLimits(props.model), [props.model])

  if (limits.length === 0) return null

  return (
    <section>
      <SectionTitle icon={Gauge}>{t('Rate limits')}</SectionTitle>
      <div className='border-border/60 overflow-hidden rounded-lg border'>
        <Table>
          <TableHeader>
            <TableRow className='bg-muted/30 hover:bg-muted/30'>
              <TableHead className='h-9 text-xs'>{t('Group')}</TableHead>
              <TableHead className='h-9 text-right text-xs'>RPM</TableHead>
              <TableHead className='h-9 text-right text-xs'>TPM</TableHead>
              <TableHead className='h-9 text-right text-xs'>RPD</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {limits.map((l) => (
              <TableRow key={l.group} className='hover:bg-muted/20'>
                <TableCell className='py-2 font-mono text-xs'>
                  {l.group}
                </TableCell>
                <TableCell className='py-2 text-right font-mono text-xs'>
                  {formatRateLimit(l.rpm)}
                </TableCell>
                <TableCell className='py-2 text-right font-mono text-xs'>
                  {formatRateLimit(l.tpm)}
                </TableCell>
                <TableCell className='py-2 text-right font-mono text-xs'>
                  {formatRateLimit(l.rpd)}
                </TableCell>
              </TableRow>
            ))}
          </TableBody>
        </Table>
      </div>
      <p className='text-muted-foreground mt-2 text-[11px] leading-relaxed'>
        {t(
          'RPM = requests per minute, TPM = tokens per minute, RPD = requests per day. Limits apply per token group.'
        )}
      </p>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Provider info card (vendor / tokenizer / license / privacy)
// ---------------------------------------------------------------------------
//
// Exported separately so the Overview tab can render it alongside capabilities
// and modalities (i.e. "what is this model?" rather than "how do I call it?").

export function ModelDetailsProviderInfo(props: { model: PricingModel }) {
  const { t } = useTranslation()
  const info = useMemo(() => inferApiInfo(props.model), [props.model])

  return (
    <section>
      <SectionTitle icon={ShieldCheck}>
        {t('Provider & data privacy')}
      </SectionTitle>

      <div className='border-border/60 bg-border/60 grid grid-cols-1 gap-px overflow-hidden rounded-lg border sm:grid-cols-2'>
        <InfoCell label={t('Provider')}>
          <div className='flex items-center gap-1.5'>
            <span className='text-sm font-medium'>{info.vendor_label}</span>
            {info.homepage && (
              <a
                href={info.homepage}
                target='_blank'
                rel='noopener noreferrer'
                className='text-muted-foreground hover:text-foreground inline-flex items-center gap-0.5 text-[11px]'
              >
                {t('Docs')}
                <ExternalLink className='size-3' />
              </a>
            )}
          </div>
        </InfoCell>

        <InfoCell label={t('Tokenizer')}>
          <div className='flex flex-col gap-0.5'>
            <code className='font-mono text-xs'>{info.tokenizer}</code>
            {info.tokenizer_note && (
              <span className='text-muted-foreground text-[10px]'>
                {info.tokenizer_note}
              </span>
            )}
          </div>
        </InfoCell>

        <InfoCell label={t('License')}>
          <div className='flex flex-col gap-1'>
            <span className='text-sm'>{info.license}</span>
            <Badge
              variant='outline'
              className={cn(
                'h-4 w-fit px-1.5 text-[9px] font-medium',
                info.license_kind === 'open' &&
                  'border-emerald-500/40 text-emerald-600 dark:text-emerald-400',
                info.license_kind === 'open-weight' &&
                  'border-sky-500/40 text-sky-600 dark:text-sky-400',
                info.license_kind === 'proprietary' &&
                  'border-amber-500/40 text-amber-600 dark:text-amber-400'
              )}
            >
              {info.license_kind === 'open'
                ? t('Open source')
                : info.license_kind === 'open-weight'
                  ? t('Open weights')
                  : info.license_kind === 'proprietary'
                    ? t('Proprietary')
                    : t('Unknown')}
            </Badge>
          </div>
        </InfoCell>

        <InfoCell label={t('Data retention')}>
          <span className='text-sm'>
            {info.data_retention_days === 0
              ? t('Zero retention')
              : `${info.data_retention_days} ${t('days')}`}
          </span>
          <span className='text-muted-foreground text-[10px]'>
            {info.training_opt_out
              ? t('Not used for upstream training by default')
              : t('May be used for training by upstream provider')}
          </span>
        </InfoCell>
      </div>
    </section>
  )
}

function InfoCell(props: { label: string; children: React.ReactNode }) {
  return (
    <div className='bg-card flex flex-col gap-1 px-3 py-2.5'>
      <span className='text-muted-foreground text-[10px] font-medium tracking-wider uppercase'>
        {props.label}
      </span>
      {props.children}
    </div>
  )
}

// ---------------------------------------------------------------------------
// Authentication preview
// ---------------------------------------------------------------------------

function AuthSection() {
  const { t } = useTranslation()
  return (
    <section>
      <SectionTitle icon={KeyRound}>{t('Authentication')}</SectionTitle>
      <div className='border-border/60 bg-muted/20 flex items-start gap-2 rounded-lg border p-3'>
        <ChevronRight className='text-muted-foreground mt-0.5 size-3.5 shrink-0' />
        <div className='space-y-1.5 text-xs leading-relaxed'>
          <p>
            {t('All requests must include')}{' '}
            <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
              Authorization: Bearer &lt;TOKEN&gt;
            </code>{' '}
            {t('header. Anthropic-formatted endpoints accept the')}{' '}
            <code className='bg-muted rounded px-1 py-0.5 font-mono text-[11px]'>
              x-api-key
            </code>{' '}
            {t('header instead.')}
          </p>
          <p className='text-muted-foreground'>
            {t(
              'Generate tokens from the Tokens page; you can scope them to specific models, groups, IPs, and rate-limits.'
            )}
          </p>
        </div>
      </div>
    </section>
  )
}

// ---------------------------------------------------------------------------
// Composite API tab
// ---------------------------------------------------------------------------

export function ModelDetailsApi(props: {
  model: PricingModel
  endpointMap: Record<string, PricingEndpoint>
}) {
  return (
    <div className='space-y-6'>
      <CodeSamplesSection model={props.model} endpointMap={props.endpointMap} />
      <AuthSection />
      <SupportedParametersSection model={props.model} />
      <RateLimitsSection model={props.model} />
    </div>
  )
}

// ---------------------------------------------------------------------------
// Local UI helpers
// ---------------------------------------------------------------------------

function SectionTitle(props: {
  children: React.ReactNode
  icon: React.ComponentType<{ className?: string }>
}) {
  const Icon = props.icon
  return (
    <h3 className='text-foreground mb-3 flex items-center gap-1.5 text-sm font-semibold'>
      <Icon className='text-muted-foreground/70 size-3.5' />
      {props.children}
    </h3>
  )
}

// Re-export so the parent can keep its own SectionTitle if it wants:
export { Zap as ApiTabIcon }
