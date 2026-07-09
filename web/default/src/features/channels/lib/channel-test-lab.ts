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

export type ChannelTestEndpointType =
  | 'auto'
  | 'openai'
  | 'openai-response'
  | 'openai-response-compact'
  | 'anthropic'
  | 'gemini'
  | 'jina-rerank'
  | 'image-generation'
  | 'openai-video'
  | 'openai-video-async'
  | 'sanbao-image'
  | 'sanbao-video'
  | 'sanbao-upload'
  | 'sanbao-image-poll'
  | 'sanbao-video-poll'
  | 'embeddings'

export type ChannelTestTemplate = {
  endpointType: ChannelTestEndpointType
  labelKey: string
  descriptionKey: string
  defaultModel: string
  supportsStream: boolean
  payload: Record<string, unknown>
}

export const CHANNEL_TEST_TEMPLATES: ChannelTestTemplate[] = [
  {
    endpointType: 'auto',
    labelKey: 'Auto detect',
    descriptionKey: 'Use the backend to infer the endpoint from the model name',
    defaultModel: '',
    supportsStream: true,
    payload: {
      model: '{{model}}',
      messages: [{ role: 'user', content: 'ping' }],
    },
  },
  {
    endpointType: 'openai',
    labelKey: 'Chat Completions',
    descriptionKey: 'Test /v1/chat/completions compatibility',
    defaultModel: 'gpt-4o-mini',
    supportsStream: true,
    payload: {
      model: '{{model}}',
      messages: [{ role: 'user', content: 'ping' }],
      max_tokens: 16,
    },
  },
  {
    endpointType: 'openai-response',
    labelKey: 'Responses',
    descriptionKey: 'Test /v1/responses compatibility',
    defaultModel: 'gpt-4o-mini',
    supportsStream: true,
    payload: {
      model: '{{model}}',
      input: 'ping',
      max_output_tokens: 16,
    },
  },
  {
    endpointType: 'openai-response-compact',
    labelKey: 'Responses Compaction',
    descriptionKey: 'Test /v1/responses/compact compatibility',
    defaultModel: 'gpt-4o-mini',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      input: 'Summarize this short test message.',
      max_output_tokens: 16,
    },
  },
  {
    endpointType: 'anthropic',
    labelKey: 'Anthropic Messages',
    descriptionKey: 'Test /v1/messages compatibility',
    defaultModel: 'claude-3-5-haiku-latest',
    supportsStream: true,
    payload: {
      model: '{{model}}',
      messages: [{ role: 'user', content: 'ping' }],
      max_tokens: 16,
    },
  },
  {
    endpointType: 'gemini',
    labelKey: 'Gemini Generate Content',
    descriptionKey: 'Test Gemini generateContent compatibility',
    defaultModel: 'gemini-2.0-flash',
    supportsStream: true,
    payload: {
      contents: [{ parts: [{ text: 'ping' }] }],
    },
  },
  {
    endpointType: 'image-generation',
    labelKey: 'Image Generation',
    descriptionKey: 'Test /v1/images/generations compatibility',
    defaultModel: 'gpt-image2',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: 'a cute cat',
      output_resolution: '1K',
      aspect_ratio: '1:1',
    },
  },
  {
    endpointType: 'openai-video',
    labelKey: 'Video Generation',
    descriptionKey: 'Test /v1/videos compatibility',
    defaultModel: 'seedance2(933)',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: 'a mountain at sunrise',
      size: '1280x720',
      seconds: '4',
    },
  },
  {
    endpointType: 'openai-video-async',
    labelKey: 'Async Video Generation',
    descriptionKey: 'Test /v1/video/async-generations compatibility',
    defaultModel: 'video-2.0-fast',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: 'a mountain at sunrise',
      size: '720x1280',
      duration: 4,
    },
  },
  {
    endpointType: 'sanbao-image',
    labelKey: 'Sanbao Image Generation',
    descriptionKey:
      'Test Sanbao image generation with aspect ratio, references, quality, and concurrency',
    defaultModel: 'gpt-image2',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: '一张未来城市夜景海报，霓虹灯，电影感，细节丰富',
      aspect_ratio: '16:9',
      images: ['https://example.com/reference.png'],
      quality: 'high',
      concurrency: 1,
    },
  },
  {
    endpointType: 'sanbao-video',
    labelKey: 'Sanbao Video Generation',
    descriptionKey:
      'Test Sanbao video generation with ratio, resolution, duration, and references',
    defaultModel: 'sd2_full',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: '@参考图片1 生成一个电影感产品短视频',
      ratio: '9:16',
      resolution: '720p',
      duration: 5,
      concurrency: 1,
      reference: 'all',
      images: ['https://example.com/reference.png'],
    },
  },
  {
    endpointType: 'sanbao-upload',
    labelKey: 'Sanbao Reference Upload',
    descriptionKey:
      'Submit a Sanbao task that uploads referenced media through the raw upload endpoint',
    defaultModel: 'sd2_full',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      prompt: '@参考图片1 使用参考素材生成一个短视频',
      ratio: '9:16',
      resolution: '720p',
      duration: 5,
      concurrency: 1,
      reference: 'all',
      images: ['https://example.com/reference.png'],
    },
  },
  {
    endpointType: 'sanbao-image-poll',
    labelKey: 'Sanbao Image Polling',
    descriptionKey: 'Poll a Sanbao image task by local task id',
    defaultModel: 'gpt-image2',
    supportsStream: false,
    payload: {
      task_id: 'task_xxx',
    },
  },
  {
    endpointType: 'sanbao-video-poll',
    labelKey: 'Sanbao Video Polling',
    descriptionKey: 'Poll a Sanbao video task by local task id',
    defaultModel: 'sd2_full',
    supportsStream: false,
    payload: {
      task_id: 'task_xxx',
    },
  },
  {
    endpointType: 'embeddings',
    labelKey: 'Embeddings',
    descriptionKey: 'Test /v1/embeddings compatibility',
    defaultModel: 'text-embedding-3-small',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      input: 'ping',
    },
  },
  {
    endpointType: 'jina-rerank',
    labelKey: 'Rerank',
    descriptionKey: 'Test /v1/rerank compatibility',
    defaultModel: 'jina-reranker-v2-base-multilingual',
    supportsStream: false,
    payload: {
      model: '{{model}}',
      query: 'ping',
      documents: ['ping', 'pong'],
    },
  },
]

export function getChannelTestTemplate(
  endpointType: ChannelTestEndpointType
): ChannelTestTemplate {
  return (
    CHANNEL_TEST_TEMPLATES.find(
      (template) => template.endpointType === endpointType
    ) ?? CHANNEL_TEST_TEMPLATES[0]
  )
}

export function getChannelTestPreviewPayload(
  endpointType: ChannelTestEndpointType,
  model: string
) {
  const template = getChannelTestTemplate(endpointType)
  return JSON.parse(
    JSON.stringify(template.payload).replaceAll(
      '{{model}}',
      model.trim() || template.defaultModel || '<model>'
    )
  ) as Record<string, unknown>
}
