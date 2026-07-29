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
  getCreationVideoCapabilities,
  getCreationVideoRequestOptions,
  type CreationVideoOptions,
  type CreationVideoReferences,
} from './session'
import type { CreationModel } from './types'

export type CreationVideoSubmitRequest = {
  endpoint: '/api/creation/video/async-generations' | '/pg/chat/completions'
  payload: Record<string, unknown>
  transport: 'async-video' | 'prompt'
}

export function buildCreationVideoSubmitRequest(params: {
  model: CreationModel
  prompt: string
  content?: unknown
  videoOptions?: CreationVideoOptions
  videoReferences?: CreationVideoReferences
}): CreationVideoSubmitRequest {
  const capability = getCreationVideoCapabilities(params.model)
  const supportsVideoEndpoint = params.model.supported_endpoint_types.some(
    (endpoint) => endpoint === 'openai-video'
  )
  if (!capability && !supportsVideoEndpoint) {
    return {
      endpoint: '/pg/chat/completions',
      payload: {
        model: params.model.id,
        messages: [
          {
            role: 'user',
            content: params.content ?? params.prompt,
          },
        ],
        stream: false,
      },
      transport: 'prompt',
    }
  }

  const basePayload = {
    model: params.model.id,
    prompt: params.prompt,
  }
  if (!capability) {
    return {
      endpoint: '/api/creation/video/async-generations',
      payload: basePayload,
      transport: 'async-video',
    }
  }

  const requestOptions = getCreationVideoRequestOptions(
    params.videoOptions ?? DEFAULT_CREATION_VIDEO_OPTIONS,
    params.model,
    params.videoReferences
  )
  const { estimateSeconds: _estimateSeconds, ...requestPayload } =
    requestOptions
  return {
    endpoint: '/api/creation/video/async-generations',
    payload: {
      ...basePayload,
      ...requestPayload,
    },
    transport: 'async-video',
  }
}

export function extractPromptVideoURL(text?: string) {
  const trimmed = text?.trim()
  if (!trimmed) return undefined

  try {
    const parsed = JSON.parse(trimmed) as unknown
    const jsonURL = findVideoURL(parsed)
    if (jsonURL) return jsonURL
  } catch {
    // The common response is plain text or Markdown rather than JSON.
  }

  const urls = trimmed
    .match(/https?:\/\/[^\s<>"'`]+/gi)
    ?.map((url) => url.replace(/[),.;\]}]+$/, ''))
    .filter(Boolean)
  if (!urls?.length) return undefined
  return (
    urls.find((url) => /\.(?:m4v|mov|mp4|webm)(?:[?#]|$)/i.test(url)) ?? urls[0]
  )
}

function findVideoURL(value: unknown, depth = 0): string | undefined {
  if (depth > 5 || !value) return undefined
  if (typeof value === 'string') {
    return /^https?:\/\//i.test(value.trim()) ? value.trim() : undefined
  }
  if (Array.isArray(value)) {
    for (const item of value) {
      const url = findVideoURL(item, depth + 1)
      if (url) return url
    }
    return undefined
  }
  if (typeof value !== 'object') return undefined

  const record = value as Record<string, unknown>
  for (const key of [
    'video_url',
    'result_url',
    'output_url',
    'content_url',
    'url',
  ]) {
    const url = findVideoURL(record[key], depth + 1)
    if (url) return url
  }
  for (const item of Object.values(record)) {
    const url = findVideoURL(item, depth + 1)
    if (url) return url
  }
  return undefined
}
