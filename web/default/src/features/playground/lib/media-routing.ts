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
  DEFAULT_CREATION_VIDEO_OPTIONS,
  getCreationVideoRequestOptions,
} from '../../creation-center/session'
import { API_ENDPOINTS, MESSAGE_ROLES } from '../constants'
import type { Message } from '../types'

export type PlaygroundMediaMode = 'chat' | 'image' | 'video'

export type PlaygroundImageRequest = {
  model: string
  prompt: string
  n: number
}

export type PlaygroundVideoRequest = {
  model: string
  prompt: string
  seconds: string
  size: string
}

export type PlaygroundMediaRequest =
  | PlaygroundImageRequest
  | PlaygroundVideoRequest

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
  return {
    model,
    prompt,
    seconds: videoOptions.seconds,
    size: videoOptions.size,
  }
}

export function formatPlaygroundMediaResult(
  raw: unknown,
  model: string
): string {
  const error = extractErrorMessage(raw)
  if (error) {
    throw new Error(error)
  }

  const mode = getPlaygroundModelMode(model)
  if (mode === 'image') {
    return formatImageResult(raw, model)
  }
  if (mode === 'video') {
    return formatVideoResult(raw, model)
  }
  return ''
}

function formatImageResult(raw: unknown, model: string) {
  const data = asRecord(raw)
  const firstImage = Array.isArray(data.data) ? asRecord(data.data[0]) : {}
  const b64 = getString(firstImage, 'b64_json')
  const imageUrl =
    getString(firstImage, 'url') ||
    (b64 ? `data:image/png;base64,${b64}` : undefined)
  const id = getString(data, 'id')
  const revisedPrompt = getString(firstImage, 'revised_prompt')

  const lines = [`图片生成完成。`, `模型：${model}`]
  if (id) lines.push(`结果 ID：${id}`)
  if (revisedPrompt) lines.push(`优化提示词：${revisedPrompt}`)
  if (imageUrl) {
    lines.push('', `![${model} 生成图](${imageUrl})`)
  } else {
    lines.push('接口已返回结果，但暂未解析到图片地址。')
  }

  return lines.join('\n')
}

function formatVideoResult(raw: unknown, model: string) {
  const data = asRecord(raw)
  const envelopeData = asRecord(data.data)
  const source = Object.keys(envelopeData).length ? envelopeData : data
  const metadata = asRecord(source.metadata)
  const taskId =
    getString(source, 'task_id') ||
    getString(data, 'task_id') ||
    getString(source, 'id') ||
    getString(data, 'id')
  const status = getString(source, 'status') || getString(data, 'status')
  const videoUrl =
    getString(source, 'url') ||
    getString(source, 'result_url') ||
    getString(source, 'output_url') ||
    getString(source, 'video_url') ||
    getString(metadata, 'url') ||
    getString(metadata, 'result_url') ||
    getString(metadata, 'output_url') ||
    getString(metadata, 'video_url')

  const lines = [`视频任务已提交。`, `模型：${model}`]
  if (taskId) lines.push(`任务 ID：${taskId}`)
  if (status) lines.push(`当前状态：${status}`)
  if (videoUrl) {
    lines.push(`视频地址：${videoUrl}`)
  } else {
    lines.push('生成完成后，可在任务日志中查看结果。')
  }

  return lines.join('\n')
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
  const data = asRecord(raw)
  const error = asRecord(data.error)
  return getString(error, 'message') || getString(data, 'message')
}

function getString(data: Record<string, unknown>, key: string) {
  const value = data[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function asRecord(value: unknown): Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}
