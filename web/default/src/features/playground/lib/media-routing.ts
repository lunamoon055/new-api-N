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
import {
  extractMediaErrorMessage,
  parseImageGenerationResult,
  parseVideoGenerationResult,
} from '@/features/media-generation/result-parsers'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  getCreationVideoRequestOptions,
  type CreationVideoRequestOptions,
} from '@/features/media-generation/video-options'
import { API_ENDPOINTS, MESSAGE_ROLES } from '../constants'
import type { Message } from '../types'

export type PlaygroundMediaMode = 'chat' | 'image' | 'video'

export type PlaygroundImageRequest = {
  model: string
  prompt: string
  n: number
}

type WithoutEstimate<T> = T extends { estimateSeconds: number }
  ? Omit<T, 'estimateSeconds'>
  : never

export type PlaygroundVideoRequest = {
  model: string
  prompt: string
} & WithoutEstimate<CreationVideoRequestOptions>

export type PlaygroundMediaRequest =
  | PlaygroundImageRequest
  | PlaygroundVideoRequest

export type PlaygroundMediaResult = {
  mode: Exclude<PlaygroundMediaMode, 'chat'>
  content: string
  taskId?: string
  status?: string
  mediaUrl?: string
}

const VIDEO_MODEL_NAMES = new Set([
  'video-2.0',
  'video-2.0-fast',
  'sora2',
  'ko3',
  'kling-v3',
])

const IMAGE_MODEL_NAMES = new Set(['gpt-image2'])

export function getPlaygroundModelMode(model: string): PlaygroundMediaMode {
  const normalizedModel = normalizeModelName(model)
  if (!normalizedModel) return 'chat'

  if (
    VIDEO_MODEL_NAMES.has(normalizedModel) ||
    normalizedModel.startsWith('video-') ||
    normalizedModel.startsWith('sora') ||
    normalizedModel.startsWith('veo') ||
    normalizedModel.includes('kling') ||
    normalizedModel.includes('grok-imagine-video')
  ) {
    return 'video'
  }

  if (
    IMAGE_MODEL_NAMES.has(normalizedModel) ||
    normalizedModel.includes('gpt-image') ||
    normalizedModel.includes('nano-banana') ||
    normalizedModel.includes('imagen')
  ) {
    return 'image'
  }

  return 'chat'
}

export function getPlaygroundMediaEndpoint(model: string): string | null {
  switch (getPlaygroundModelMode(model)) {
    case 'image':
      return API_ENDPOINTS.IMAGE_GENERATIONS
    case 'video':
      return API_ENDPOINTS.VIDEO_ASYNC_GENERATIONS
    default:
      return null
  }
}

export function buildPlaygroundMediaRequest(
  model: string,
  messages: Message[]
): PlaygroundMediaRequest | null {
  const mode = getPlaygroundModelMode(model)
  if (mode === 'chat') return null

  const prompt = getLatestUserPrompt(messages)
  if (mode === 'image') {
    return {
      model,
      prompt,
      n: 1,
    }
  }

  const videoOptions = getCreationVideoRequestOptions(
    DEFAULT_CREATION_VIDEO_OPTIONS,
    model
  )
  const { estimateSeconds: _estimateSeconds, ...videoPayload } = videoOptions
  return {
    model,
    prompt,
    ...videoPayload,
  }
}

export function formatPlaygroundMediaResult(
  raw: unknown,
  model: string
): string {
  return parsePlaygroundMediaResult(raw, model).content
}

export function parsePlaygroundMediaResult(
  raw: unknown,
  model: string
): PlaygroundMediaResult {
  const error = extractErrorMessage(raw)
  if (error) {
    throw new Error(error)
  }

  const mode = getPlaygroundModelMode(model)
  if (mode === 'image') {
    return parseImageResult(raw, model)
  }
  if (mode === 'video') {
    return parseVideoResult(raw, model)
  }
  throw new Error('Current model does not support media generation')
}

export function buildPlaygroundVideoProxyUrl(taskId: string) {
  return `/v1/videos/${encodeURIComponent(taskId)}/content`
}

function normalizePreviewUrl(url: string | undefined) {
  const trimmed = url?.trim()
  if (!trimmed) return undefined
  if (
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://') ||
    trimmed.startsWith('data:') ||
    trimmed.startsWith('/')
  ) {
    return trimmed
  }
  return undefined
}

function isVideoApiContentUrl(url: string | undefined) {
  if (!url) return false
  return (
    url.includes('/v1/videos/') || url.includes('/v1/video/async-generations/')
  )
}

function parseImageResult(raw: unknown, model: string): PlaygroundMediaResult {
  const result = parseImageGenerationResult(raw)

  const lines = [`图片生成完成。`, `模型：${model}`]
  if (result.id) lines.push(`结果 ID：${result.id}`)
  if (result.revisedPrompt) lines.push(`优化提示词：${result.revisedPrompt}`)
  if (result.imageUrl) {
    lines.push('图片预览已生成。')
  } else {
    lines.push('接口已返回结果，但暂未解析到图片地址。')
  }

  return {
    mode: 'image',
    content: lines.join('\n'),
    taskId: result.id,
    mediaUrl: result.imageUrl,
  }
}

function parseVideoResult(raw: unknown, model: string): PlaygroundMediaResult {
  const result = parseVideoGenerationResult(raw)
  const normalizedResultUrl = normalizePreviewUrl(result.videoUrl)
  const videoUrl =
    normalizedResultUrl && !isVideoApiContentUrl(normalizedResultUrl)
      ? normalizedResultUrl
      : result.taskId
        ? buildPlaygroundVideoProxyUrl(result.taskId)
        : normalizedResultUrl

  const completed = result.status === 'completed' && !!videoUrl
  const lines = [
    completed ? `视频生成完成。` : `视频任务已提交。`,
    `模型：${model}`,
  ]
  if (result.taskId) lines.push(`任务 ID：${result.taskId}`)
  if (result.upstreamStatus) lines.push(`当前状态：${result.upstreamStatus}`)
  if (completed) {
    lines.push('视频预览已生成。')
  } else {
    lines.push('生成完成后，可在任务日志中查看结果。')
  }

  return {
    mode: 'video',
    content: lines.join('\n'),
    taskId: result.taskId,
    status: result.upstreamStatus,
    mediaUrl: completed ? videoUrl : undefined,
  }
}

function getLatestUserPrompt(messages: Message[]) {
  for (let index = messages.length - 1; index >= 0; index -= 1) {
    const message = messages[index]
    if (message?.from !== MESSAGE_ROLES.USER) continue
    const content = message.versions[0]?.content?.trim()
    if (content) return content
  }
  return ''
}

function normalizeModelName(model: string) {
  return model.trim().toLowerCase()
}

function extractErrorMessage(raw: unknown): string | undefined {
  return extractMediaErrorMessage(raw)
}
