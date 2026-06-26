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
  asRecord,
  extractMediaErrorMessage,
  getString,
  parseImageGenerationResult,
  parseVideoGenerationResult,
} from '@/features/media-generation/result-parsers'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  composeCreationPrompt,
  getCreationImageRequestOptions,
  getCreationVideoRequestOptions,
  type CreationImageReferences,
  type CreationVideoReferences,
  type CreationVideoOptions,
} from './session'
import type {
  CreationAsset,
  CreationCatalogResponse,
  CreationModelCategories,
  CreationModelDescriptions,
  CreationMode,
  CreationModel,
  CreationResult,
} from './types'

const CREATION_MODEL_CATEGORIES_OPTION_KEY = 'CreationModelCategories'
const CREATION_MODEL_DESCRIPTIONS_OPTION_KEY = 'CreationModelDescriptions'

type UpdateOptionResponse = {
  success: boolean
  message: string
}

type CreationReferenceFileUploadResponse = {
  success: boolean
  message?: string
  data?: {
    kind?: string
    mime_type?: string
    url?: string
  }
}

export async function getCreationCatalog(): Promise<CreationCatalogResponse> {
  const response = await api.get<CreationCatalogResponse>(
    '/api/creation/models'
  )
  return response.data
}

export async function saveCreationModelCategories(
  categories: CreationModelCategories
): Promise<UpdateOptionResponse> {
  const response = await api.put<UpdateOptionResponse>('/api/option/', {
    key: CREATION_MODEL_CATEGORIES_OPTION_KEY,
    value: JSON.stringify(categories),
  })
  return response.data
}

export async function saveCreationModelDescriptions(
  descriptions: CreationModelDescriptions
): Promise<UpdateOptionResponse> {
  const response = await api.put<UpdateOptionResponse>('/api/option/', {
    key: CREATION_MODEL_DESCRIPTIONS_OPTION_KEY,
    value: JSON.stringify(descriptions),
  })
  return response.data
}

export async function uploadCreationReferenceFile(
  file: File,
  kind: 'image' | 'video' | 'audio',
  mimeType?: string
) {
  const formData = new FormData()
  formData.append('kind', kind)
  formData.append(
    'file',
    mimeType && file.type !== mimeType
      ? file.slice(0, file.size, mimeType)
      : file,
    file.name
  )
  const response = await api.post<CreationReferenceFileUploadResponse>(
    '/api/creation/reference-files',
    formData,
    { skipErrorHandler: true } as Record<string, unknown>
  )
  const url = response.data.data?.url
  if (!response.data.success || !url) {
    throw new Error(response.data.message || 'Unable to upload reference file.')
  }
  return url
}

export async function submitCreationTask(params: {
  mode: CreationMode
  model: CreationModel
  prompt: string
  assets?: CreationAsset[]
  imageReferences?: CreationImageReferences
  videoOptions?: CreationVideoOptions
  videoReferences?: CreationVideoReferences
}): Promise<CreationResult> {
  const promptWithAssets = composeCreationPrompt(
    params.prompt,
    params.assets ?? []
  )
  if (params.mode === 'chat') {
    const response = await api.post(
      '/pg/chat/completions',
      {
        model: params.model.id,
        messages: [
          {
            role: 'user',
            content: buildChatContent(promptWithAssets, params.assets ?? []),
          },
        ],
        stream: false,
      },
      { skipErrorHandler: true } as Record<string, unknown>
    )
    return parseChatResult(response.data, params.model.id)
  }

  if (params.mode === 'image') {
    const imageOptions = getCreationImageRequestOptions(
      promptWithAssets,
      params.model.id,
      params.imageReferences
    )
    const response = await api.post(
      '/api/creation/images/generations',
      {
        model: params.model.id,
        prompt: promptWithAssets,
        n: 1,
        ...imageOptions,
      },
      { skipErrorHandler: true } as Record<string, unknown>
    )
    return parseImageResult(response.data, params.model.id)
  }

  const videoOptions = getCreationVideoRequestOptions(
    params.videoOptions ?? DEFAULT_CREATION_VIDEO_OPTIONS,
    params.model.id,
    params.videoReferences
  )
  const { estimateSeconds: _estimateSeconds, ...videoPayload } = videoOptions
  const response = await api.post(
    '/api/creation/video/async-generations',
    {
      model: params.model.id,
      prompt: promptWithAssets,
      ...videoPayload,
    },
    { skipErrorHandler: true } as Record<string, unknown>
  )
  return parseVideoResult(response.data, params.model.id)
}

function buildChatContent(prompt: string, assets: CreationAsset[]) {
  const imageAssets = assets.filter(
    (asset) => asset.type.startsWith('image/') && asset.dataUrl
  )
  if (!imageAssets.length) return prompt

  return [
    { type: 'text', text: prompt },
    ...imageAssets.map((asset) => ({
      type: 'image_url',
      image_url: { url: asset.dataUrl },
    })),
  ]
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

  const result = parseImageGenerationResult(raw)

  return {
    mode: 'image',
    model,
    id: result.id,
    status: result.imageUrl ? 'completed' : 'unknown',
    imageUrl: result.imageUrl,
    outputText: result.revisedPrompt,
    raw,
  }
}

function parseVideoResult(raw: unknown, model: string): CreationResult {
  const error = extractErrorMessage(raw)
  if (error) {
    return { mode: 'video', model, status: 'failed', error, raw }
  }

  const result = parseVideoGenerationResult(raw)

  return {
    mode: 'video',
    model,
    id: result.id,
    taskId: result.taskId,
    status: result.status,
    videoUrl: result.videoUrl,
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
  return extractMediaErrorMessage(raw)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}
