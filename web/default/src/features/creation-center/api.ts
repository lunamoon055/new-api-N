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
import { api } from '@/lib/api'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  getCreationVideoRequestOptions,
  type CreationVideoOptions,
} from './session'
import type {
  CreationCatalogResponse,
  CreationMode,
  CreationModel,
  CreationResult,
  CreationResultStatus,
} from './types'

export async function getCreationCatalog(): Promise<CreationCatalogResponse> {
  const response = await api.get<CreationCatalogResponse>(
    '/api/creation/models'
  )
  return response.data
}

export async function submitCreationTask(params: {
  mode: CreationMode
  model: CreationModel
  prompt: string
  videoOptions?: CreationVideoOptions
}): Promise<CreationResult> {
  if (params.mode === 'chat') {
    const response = await api.post(
      '/pg/chat/completions',
      {
        model: params.model.id,
        messages: [{ role: 'user', content: params.prompt }],
        stream: false,
      },
      { skipErrorHandler: true } as Record<string, unknown>
    )
    return parseChatResult(response.data, params.model.id)
  }

  if (params.mode === 'image') {
    const response = await api.post(
      '/api/creation/images/generations',
      {
        model: params.model.id,
        prompt: params.prompt,
        n: 1,
      },
      { skipErrorHandler: true } as Record<string, unknown>
    )
    return parseImageResult(response.data, params.model.id)
  }

  const videoOptions = getCreationVideoRequestOptions(
    params.videoOptions ?? DEFAULT_CREATION_VIDEO_OPTIONS,
    params.model.id
  )
  const response = await api.post(
    '/api/creation/video/async-generations',
    {
      model: params.model.id,
      prompt: params.prompt,
      seconds: videoOptions.seconds,
      size: videoOptions.size,
    },
    { skipErrorHandler: true } as Record<string, unknown>
  )
  return parseVideoResult(response.data, params.model.id)
}

export async function getCreationVideoTask(params: {
  taskId: string
  model: string
}): Promise<CreationResult> {
  const response = await api.get(
    `/api/creation/video/async-generations/${encodeURIComponent(params.taskId)}`,
    { skipErrorHandler: true, disableDuplicate: true } as Record<
      string,
      unknown
    >
  )
  return parseVideoResult(response.data, params.model)
}

export function getCreationErrorMessage(error: unknown): string {
  if (isRecord(error)) {
    const response = error.response
    if (isRecord(response)) {
      const message = extractErrorMessage(response.data)
      if (message) return message
    }
    if (typeof error.message === 'string') return error.message
  }
  return 'Request failed'
}

function parseChatResult(raw: unknown, model: string): CreationResult {
  const error = extractErrorMessage(raw)
  if (error) {
    return { mode: 'chat', model, status: 'failed', error, raw }
  }

  const data = asRecord(raw)
  const text = extractChatText(data)
  return {
    mode: 'chat',
    model,
    id: getString(data, 'id'),
    status: text ? 'completed' : 'unknown',
    outputText: text,
    raw,
  }
}

function parseImageResult(raw: unknown, model: string): CreationResult {
  const error = extractErrorMessage(raw)
  if (error) {
    return { mode: 'image', model, status: 'failed', error, raw }
  }

  const data = asRecord(raw)
  const firstImage = Array.isArray(data.data) ? asRecord(data.data[0]) : {}
  const b64 = getString(firstImage, 'b64_json')
  const imageUrl =
    getString(firstImage, 'url') ||
    (b64 ? `data:image/png;base64,${b64}` : undefined)

  return {
    mode: 'image',
    model,
    id: getString(data, 'id'),
    status: imageUrl ? 'completed' : 'unknown',
    imageUrl,
    outputText: getString(firstImage, 'revised_prompt'),
    raw,
  }
}

function parseVideoResult(raw: unknown, model: string): CreationResult {
  const error = extractErrorMessage(raw)
  if (error) {
    return { mode: 'video', model, status: 'failed', error, raw }
  }

  const data = asRecord(raw)
  const envelopeData = asRecord(data.data)
  const source = Object.keys(envelopeData).length ? envelopeData : data
  const metadata = asRecord(source.metadata)
  const taskId =
    getString(source, 'task_id') ||
    getString(data, 'task_id') ||
    getString(source, 'id') ||
    getString(data, 'id')

  return {
    mode: 'video',
    model,
    id: getString(source, 'id') || getString(data, 'id'),
    taskId,
    status: normalizeStatus(getString(source, 'status')),
    videoUrl:
      getString(source, 'url') ||
      getString(source, 'result_url') ||
      getString(source, 'output_url') ||
      getString(source, 'video_url') ||
      getString(metadata, 'url') ||
      getString(metadata, 'result_url') ||
      getString(metadata, 'output_url') ||
      getString(metadata, 'video_url'),
    raw,
  }
}

function extractChatText(data: Record<string, unknown>): string | undefined {
  const choices = data.choices
  if (!Array.isArray(choices) || choices.length === 0) return undefined

  const firstChoice = asRecord(choices[0])
  const message = asRecord(firstChoice.message)
  const content = message.content
  if (typeof content === 'string') return content
  if (Array.isArray(content)) {
    return content
      .map((part) => getString(asRecord(part), 'text'))
      .filter(Boolean)
      .join('\n')
  }
  return getString(firstChoice, 'text')
}

function extractErrorMessage(raw: unknown): string | undefined {
  const data = asRecord(raw)
  const error = asRecord(data.error)
  return getString(error, 'message') || getString(data, 'message')
}

function normalizeStatus(status: string | undefined): CreationResultStatus {
  switch (status?.toLowerCase()) {
    case 'queued':
    case 'pending':
    case 'submitted':
      return 'queued'
    case 'processing':
    case 'running':
    case 'in_progress':
      return 'processing'
    case 'completed':
    case 'succeeded':
    case 'success':
      return 'completed'
    case 'failed':
    case 'cancelled':
    case 'canceled':
      return 'failed'
    default:
      return 'unknown'
  }
}

function getString(data: Record<string, unknown>, key: string) {
  const value = data[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

function asRecord(value: unknown): Record<string, unknown> {
  return isRecord(value) ? value : {}
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
