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
  EMPTY_CREATION_IMAGE_REFERENCES,
  getCreationImageRequestOptions,
  usesAsyncCreationImageModel,
} from '@/features/media-generation/image-options'
import {
  extractMediaErrorMessage,
  parseImageGenerationResult,
  parseVideoGenerationResult,
} from '@/features/media-generation/result-parsers'
import {
  DEFAULT_CREATION_VIDEO_OPTIONS,
  getCreationVideoCapabilities,
  getCreationDurationOptions,
  getCreationResolutionOptions,
  getCreationVideoRequestOptions,
  type CreationResolution,
  type CreationVideoRequestOptions,
} from '@/features/media-generation/video-options'
import { getXJZVideoModelSpec } from '@/features/media-generation/xjz-video-models'
import { t } from 'i18next'
import { API_ENDPOINTS, MESSAGE_ROLES } from '../constants'
import type { Message, PlaygroundModelInfo } from '../types'
import {
  buildPlaygroundReferences,
  getPlaygroundIntegration,
  resolveIntegrationChoice,
  type PlaygroundReferenceSelection,
} from './model-integration'

export type PlaygroundMediaMode = 'chat' | 'image' | 'video'
export type PlaygroundModeSelection = PlaygroundMediaMode | 'auto'
export type PlaygroundImageEndpoint = 'auto' | 'sync' | 'async'
export type PlaygroundVideoEndpoint = 'auto' | 'standard' | 'async'
export type PlaygroundVideoSelection = {
  aspectRatio: string
  duration: string
  resolution?: string
}

const FALLBACK_VIDEO_RATIOS = ['16:9', '9:16', '1:1']
const SD25_VIDEO_RATIOS = [...FALLBACK_VIDEO_RATIOS, '4:3', '3:4']
const SD25_VIDEO_DURATIONS = Array.from({ length: 27 }, (_, index) =>
  String(index + 4)
)
const FALLBACK_VIDEO_SIZES: Record<string, string> = {
  '16:9': '1920x1080',
  '9:16': '1080x1920',
  '1:1': '1080x1080',
  '4:3': '1440x1080',
  '3:4': '1080x1440',
}

export function getPlaygroundVideoChoices(
  model: string,
  info?: PlaygroundModelInfo
) {
  const parameters = getPlaygroundIntegration(info)?.parameters
  if (parameters?.durations?.length) {
    return {
      aspectRatios: parameters.ratios ?? FALLBACK_VIDEO_RATIOS,
      durations: parameters.durations.map(String),
      resolutions: parameters.resolutions ?? [],
    }
  }
  const capability = getCreationVideoCapabilities(model)
  if (capability) {
    return {
      aspectRatios: capability.aspectRatios,
      durations: getCreationDurationOptions(model).map(
        (option) => option.value
      ),
      resolutions: getCreationResolutionOptions(model).map(
        (option) => option.value
      ),
    }
  }

  // These gateway model names are outside the creation-center catalog.
  const normalizedModel = model.toLowerCase()
  if (normalizedModel.includes('sd2.5') || normalizedModel === 'sd-2.5') {
    return {
      aspectRatios: SD25_VIDEO_RATIOS,
      durations: SD25_VIDEO_DURATIONS,
      resolutions: [],
    }
  }
  if (normalizedModel.startsWith('omni-video-1')) {
    return {
      aspectRatios: FALLBACK_VIDEO_RATIOS,
      durations: ['3', '5', '8'],
      resolutions: [],
    }
  }
  return {
    aspectRatios: FALLBACK_VIDEO_RATIOS,
    durations: getCreationDurationOptions(model).map((option) => option.value),
    resolutions: [],
  }
}

export function resolvePlaygroundVideoSelection(
  model: string,
  selection: PlaygroundVideoSelection,
  info?: PlaygroundModelInfo
): PlaygroundVideoSelection {
  const choices = getPlaygroundVideoChoices(model, info)
  const preferredDuration =
    selection.duration || DEFAULT_CREATION_VIDEO_OPTIONS.duration
  const resolved = {
    aspectRatio: choices.aspectRatios.includes(selection.aspectRatio)
      ? selection.aspectRatio
      : choices.aspectRatios[0],
    duration: choices.durations.includes(preferredDuration)
      ? preferredDuration
      : choices.durations[0],
  }
  const spec = getXJZVideoModelSpec(model)
  if (
    getPlaygroundIntegration(info)?.parameters ||
    (selection.resolution && choices.resolutions.length)
  ) {
    return {
      ...resolved,
      resolution: resolveIntegrationChoice(
        selection.resolution,
        choices.resolutions
      ),
    }
  }
  if (spec?.selectableResolution) {
    return {
      ...resolved,
      resolution: spec.resolutions.includes(
        selection.resolution as CreationResolution
      )
        ? selection.resolution
        : spec.resolutions[0],
    }
  }
  return resolved
}

export type PlaygroundImageRequest = {
  model: string
  prompt: string
  n?: number
} & Record<string, unknown>

type WithoutEstimate<T> = T extends { estimateSeconds: number }
  ? Omit<T, 'estimateSeconds'>
  : never

export type PlaygroundVideoRequest = {
  model: string
  prompt: string
} & (WithoutEstimate<CreationVideoRequestOptions> | Record<string, unknown>)

export type PlaygroundMediaRequest =
  PlaygroundImageRequest | PlaygroundVideoRequest

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
  'video-2.0-mini',
  'video-2.0-480p',
  'video-2.0-fast-480p',
  'video-2.0-mini-480p',
  'sora2',
  'videos-standard',
  'videos-fast',
  'videos-mini',
  'sd2-mini',
  'sd2-fast',
  'sd2满血',
  'sd-2.0-933',
  'ko3',
  'kling-v3',
])

const IMAGE_MODEL_NAMES = new Set(['gpt-image2', 'seedream-5-0'])

export function getPlaygroundModelMode(
  model: string,
  info?: PlaygroundModelInfo
): PlaygroundMediaMode {
  const integration = getPlaygroundIntegration(info)
  if (integration?.kind === 'image' || integration?.kind === 'video')
    return integration.kind
  if (getXJZVideoModelSpec(model)) return 'video'
  const normalizedModel = normalizeModelName(model)
  if (!normalizedModel) return 'chat'

  if (
    VIDEO_MODEL_NAMES.has(normalizedModel) ||
    normalizedModel.startsWith('video-') ||
    normalizedModel.startsWith('videos-') ||
    normalizedModel.startsWith('sd2-') ||
    normalizedModel.startsWith('sd-2.0-') ||
    normalizedModel.startsWith('sd-2.5') ||
    normalizedModel.includes('sd2.5') ||
    normalizedModel.startsWith('sora') ||
    normalizedModel.startsWith('veo') ||
    normalizedModel.includes('kling') ||
    normalizedModel.includes('grok-imagine-video') ||
    normalizedModel.startsWith('omni-video-')
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

export function resolvePlaygroundMode(
  model: string,
  mode: PlaygroundModeSelection,
  info?: PlaygroundModelInfo
): PlaygroundMediaMode {
  return mode === 'auto' ? getPlaygroundModelMode(model, info) : mode
}

export function parsePlaygroundExtraBody(
  value: string
): Record<string, unknown> {
  if (!value.trim()) return {}
  let parsed: unknown
  try {
    parsed = JSON.parse(value) as unknown
  } catch {
    throw new Error(t('Extra parameters must be valid JSON'))
  }
  if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
    throw new Error(t('Extra parameters must be a JSON object'))
  }
  return parsed as Record<string, unknown>
}

export function getPlaygroundMediaEndpoint(
  model: string,
  mode: PlaygroundModeSelection = 'auto',
  imageEndpoint: PlaygroundImageEndpoint = 'auto',
  videoEndpoint: PlaygroundVideoEndpoint = 'auto',
  info?: PlaygroundModelInfo
): string | null {
  const resolvedMode = resolvePlaygroundMode(model, mode, info)
  const integration = getPlaygroundIntegration(info)
  if (integration?.kind === resolvedMode) {
    if (resolvedMode === 'image' && imageEndpoint === 'auto') {
      return integration.path?.includes('async-generations')
        ? API_ENDPOINTS.IMAGE_ASYNC_GENERATIONS
        : API_ENDPOINTS.IMAGE_GENERATIONS
    }
    if (resolvedMode === 'video' && videoEndpoint === 'auto') {
      return integration.path?.includes('async-generations')
        ? API_ENDPOINTS.VIDEO_ASYNC_GENERATIONS
        : API_ENDPOINTS.VIDEO_GENERATIONS
    }
  }
  switch (resolvedMode) {
    case 'image':
      return (
        imageEndpoint === 'auto'
          ? usesAsyncCreationImageModel(model)
          : imageEndpoint === 'async'
      )
        ? API_ENDPOINTS.IMAGE_ASYNC_GENERATIONS
        : API_ENDPOINTS.IMAGE_GENERATIONS
    case 'video':
      if (getXJZVideoModelSpec(model)) return API_ENDPOINTS.VIDEO_GENERATIONS
      return videoEndpoint === 'standard'
        ? API_ENDPOINTS.VIDEO_GENERATIONS
        : API_ENDPOINTS.VIDEO_ASYNC_GENERATIONS
    default:
      return null
  }
}

export function buildPlaygroundMediaRequest(
  model: string,
  messages: Message[],
  mode: PlaygroundModeSelection = 'auto',
  imageEndpoint: PlaygroundImageEndpoint = 'auto',
  extraBody: Record<string, unknown> = {},
  videoSelection: PlaygroundVideoSelection = { aspectRatio: '', duration: '' },
  info?: PlaygroundModelInfo,
  mediaSelection: PlaygroundReferenceSelection = {}
): PlaygroundMediaRequest | null {
  const resolvedMode = resolvePlaygroundMode(model, mode, info)
  if (resolvedMode === 'chat') return null

  const prompt = getLatestUserPrompt(messages)
  const integration = getPlaygroundIntegration(info)
  const parameters =
    integration?.kind === resolvedMode ? integration.parameters : undefined
  if (parameters) {
    const payload: Record<string, unknown> = { ...extraBody }
    const imageIsAsync =
      resolvedMode === 'image' &&
      getPlaygroundMediaEndpoint(model, mode, imageEndpoint, 'auto', info) ===
        API_ENDPOINTS.IMAGE_ASYNC_GENERATIONS
    const contract = imageIsAsync
      ? {
          ...parameters,
          resolution_field: 'output_resolution',
          image_field: 'image_urls',
        }
      : parameters
    const selection =
      resolvedMode === 'video'
        ? resolvePlaygroundVideoSelection(model, videoSelection, info)
        : {
            aspectRatio: resolveIntegrationChoice(
              mediaSelection.aspectRatio,
              parameters.ratios
            ),
            resolution: resolveIntegrationChoice(
              mediaSelection.resolution,
              parameters.resolutions
            ),
          }
    if (resolvedMode === 'image') payload.n = 1
    if (parameters.duration_field && resolvedMode === 'video')
      payload[parameters.duration_field] = Number(
        (selection as PlaygroundVideoSelection).duration
      )
    if (parameters.ratio_field && selection.aspectRatio)
      payload[parameters.ratio_field] = selection.aspectRatio
    if (parameters.resolution_field && selection.resolution)
      payload[contract.resolution_field ?? parameters.resolution_field] =
        parameters.request_resolution ?? selection.resolution
    const readExtraReferences = (field?: string) => {
      const value = field ? extraBody[field] : undefined
      return Array.isArray(value) &&
        value.every((url) => typeof url === 'string')
        ? value.join('\n')
        : undefined
    }
    Object.assign(
      payload,
      buildPlaygroundReferences(contract, {
        ...mediaSelection,
        images:
          mediaSelection.images || readExtraReferences(contract.image_field),
        videos:
          mediaSelection.videos || readExtraReferences(contract.video_field),
        audios:
          mediaSelection.audios || readExtraReferences(contract.audio_field),
      })
    )
    return { ...payload, model, prompt }
  }
  if (resolvedMode === 'image') {
    const isAsync =
      getPlaygroundMediaEndpoint(model, mode, imageEndpoint, 'auto', info) ===
      API_ENDPOINTS.IMAGE_ASYNC_GENERATIONS
    if (isAsync) {
      return {
        ...getCreationImageRequestOptions(
          prompt,
          model,
          EMPTY_CREATION_IMAGE_REFERENCES
        ),
        ...extraBody,
        model,
        prompt,
      }
    }
    return {
      n: 1,
      ...extraBody,
      model,
      prompt,
    }
  }

  const selectedVideo = resolvePlaygroundVideoSelection(
    model,
    videoSelection,
    info
  )
  const videoOptions = getCreationVideoRequestOptions(
    {
      ...DEFAULT_CREATION_VIDEO_OPTIONS,
      duration: selectedVideo.duration,
      aspectRatio: selectedVideo.aspectRatio,
      resolution:
        (selectedVideo.resolution as CreationResolution | undefined) ??
        DEFAULT_CREATION_VIDEO_OPTIONS.resolution,
    },
    model
  )
  const { estimateSeconds: _estimateSeconds, ...videoPayload } = videoOptions
  if (!getCreationVideoCapabilities(model)) {
    return {
      ...videoPayload,
      seconds: selectedVideo.duration,
      aspect_ratio: selectedVideo.aspectRatio,
      size: FALLBACK_VIDEO_SIZES[selectedVideo.aspectRatio],
      ...extraBody,
      model,
      prompt,
    }
  }
  return {
    ...videoPayload,
    ...extraBody,
    model,
    prompt,
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
  model: string,
  mode: PlaygroundModeSelection = 'auto'
): PlaygroundMediaResult {
  const error = extractErrorMessage(raw)
  if (error) {
    throw new Error(error)
  }

  const resolvedMode = resolvePlaygroundMode(model, mode)
  if (resolvedMode === 'image') {
    return parseImageResult(raw, model)
  }
  if (resolvedMode === 'video') {
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
  if (result.status === 'failed') throw new Error(t('Media generation failed'))

  const completed = !!result.imageUrl
  const lines = [
    completed ? `图片生成完成。` : `图片任务已提交。`,
    `模型：${model}`,
  ]
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
    taskId: result.taskId,
    status: result.status,
    mediaUrl: result.imageUrl,
  }
}

function parseVideoResult(raw: unknown, model: string): PlaygroundMediaResult {
  const result = parseVideoGenerationResult(raw)
  if (result.status === 'failed') throw new Error(t('Media generation failed'))
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
    status: result.status,
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
  return model
    .trim()
    .toLowerCase()
    .replace(/^\s*[(（][^()（）]*[)）]\s*/u, '')
    .trim()
}

function extractErrorMessage(raw: unknown): string | undefined {
  return extractMediaErrorMessage(raw)
}
