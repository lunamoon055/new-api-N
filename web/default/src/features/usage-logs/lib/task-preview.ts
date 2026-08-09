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
import { TASK_ACTIONS, TASK_STATUS } from '../constants'
import type { TaskLog } from '../types'

const VIDEO_ACTIONS = new Set<string>([
  TASK_ACTIONS.GENERATE,
  TASK_ACTIONS.TEXT_GENERATE,
  TASK_ACTIONS.FIRST_TAIL_GENERATE,
  TASK_ACTIONS.REFERENCE_GENERATE,
  TASK_ACTIONS.REMIX_GENERATE,
])

export type TaskInputMaterialKind = 'image' | 'video' | 'audio'

export interface TaskInputMaterial {
  kind: TaskInputMaterialKind
  url: string
}

const TASK_INPUT_KEYS: Record<TaskInputMaterialKind, string[]> = {
  image: [
    'input_images',
    'image',
    'images',
    'image_url',
    'image_urls',
    'input_reference',
    'referenceImages',
    'start_image_url',
    'end_image_url',
  ],
  video: [
    'input_videos',
    'video_url',
    'videos',
    'video_reference',
    'referenceVideos',
  ],
  audio: ['input_audios', 'audio_url', 'audios', 'referenceAudios'],
}

export function isTaskLogVideoTask(log: Pick<TaskLog, 'action'>) {
  return VIDEO_ACTIONS.has(log.action)
}

export function getTaskLogVideoPreviewUrl(
  log: Pick<TaskLog, 'action' | 'status' | 'task_id' | 'result_url' | 'data'>
) {
  if (log.status !== TASK_STATUS.SUCCESS || !isTaskLogVideoTask(log)) {
    return null
  }

  const resultUrl = normalizePreviewUrl(log.result_url)
  if (resultUrl && !isVideoApiContentUrl(resultUrl, log.task_id)) {
    return resultUrl
  }

  const dataUrl = findFirstVideoUrlInTaskData(log.data, log.task_id)
  if (dataUrl && !isVideoApiContentUrl(dataUrl, log.task_id)) {
    return dataUrl
  }

  if (log.task_id) {
    return `/v1/videos/${encodeURIComponent(log.task_id)}/content`
  }

  return resultUrl
}

export function getTaskLogPrompt(
  log: Pick<TaskLog, 'prompt' | 'properties' | 'data'>
) {
  const explicitPrompt = normalizePrompt(log.prompt)
  if (explicitPrompt) return explicitPrompt

  const propertiesPrompt = findFirstPrompt(log.properties)
  if (propertiesPrompt) return propertiesPrompt

  return findFirstPrompt(log.data)
}

export function getTaskLogModelName(
  log: Pick<TaskLog, 'model_name' | 'properties'>
) {
  const explicitModelName = normalizePrompt(log.model_name)
  if (explicitModelName) return explicitModelName

  const properties = parseJsonRecord(log.properties)
  if (!properties) return null

  return (
    normalizePrompt(properties.origin_model_name) ??
    normalizePrompt(properties.upstream_model_name)
  )
}

export function getTaskLogInputMaterials(
  log: Pick<TaskLog, 'properties'>
): TaskInputMaterial[] {
  const properties = parseJsonRecord(log.properties)
  if (!properties) return []

  const result: TaskInputMaterial[] = []
  const seen = new Set<string>()
  const collect = (record: Record<string, unknown>) => {
    for (const kind of ['image', 'video', 'audio'] as const) {
      for (const key of TASK_INPUT_KEYS[kind]) {
        for (const url of findMaterialUrls(record[key])) {
          const identity = `${kind}:${url}`
          if (seen.has(identity)) continue
          seen.add(identity)
          result.push({ kind, url })
        }
      }
    }
  }

  collect(properties)
  for (const key of ['request', 'body', 'payload']) {
    const nested = parseJsonRecord(properties[key])
    if (nested) collect(nested)
  }

  return result
}

export function getVisibleTaskLogInputMaterials(
  log: Pick<TaskLog, 'properties'>,
  canViewInputMaterials: boolean
): TaskInputMaterial[] {
  return canViewInputMaterials ? getTaskLogInputMaterials(log) : []
}

function normalizePreviewUrl(url: string | undefined) {
  const trimmed = url?.trim()
  if (!trimmed) return null
  if (
    trimmed.startsWith('http://') ||
    trimmed.startsWith('https://') ||
    trimmed.startsWith('data:') ||
    trimmed.startsWith('/')
  ) {
    return trimmed
  }
  return null
}

function isVideoApiContentUrl(url: string, taskId?: string) {
  if (taskId && url.includes(`/v1/videos/${taskId}/content`)) {
    return true
  }
  return (
    isSameSiteVideoProxyUrl(url) || url.includes('/v1/video/async-generations/')
  )
}

function isSameSiteVideoProxyUrl(url: string) {
  if (url.startsWith('/v1/videos/')) {
    return true
  }
  if (typeof window === 'undefined') {
    return false
  }
  try {
    const parsed = new URL(url)
    return (
      parsed.origin === window.location.origin &&
      parsed.pathname.startsWith('/v1/videos/')
    )
  } catch {
    return false
  }
}

function findFirstVideoUrlInTaskData(data: unknown, taskId?: string) {
  if (typeof data !== 'string') {
    return findFirstVideoUrl(data, taskId)
  }

  if (!data.trim()) return null

  try {
    return findFirstVideoUrl(JSON.parse(data), taskId)
  } catch {
    return null
  }
}

function findFirstVideoUrl(value: unknown, taskId?: string): string | null {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    const normalized = normalizePreviewUrl(trimmed)
    if (normalized) {
      return isVideoApiContentUrl(normalized, taskId) ? null : normalized
    }

    if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
      try {
        return findFirstVideoUrl(JSON.parse(trimmed), taskId)
      } catch {
        return null
      }
    }
    return null
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      const url = findFirstVideoUrl(item, taskId)
      if (url) return url
    }
    return null
  }

  if (!value || typeof value !== 'object') return null

  const record = value as Record<string, unknown>
  for (const key of [
    'video_url',
    'url',
    'result_url',
    'output_url',
    'download_url',
  ]) {
    const url = findFirstVideoUrl(record[key], taskId)
    if (url) return url
  }

  for (const key of [
    'metadata',
    'result',
    'response',
    'data',
    'outputs',
    'output',
    'results',
    'videos',
    'urls',
    'video_urls',
  ]) {
    const url = findFirstVideoUrl(record[key], taskId)
    if (url) return url
  }

  return null
}

function normalizePrompt(value: unknown) {
  return typeof value === 'string' && value.trim() ? value.trim() : null
}

function parseJsonRecord(value: unknown): Record<string, unknown> | null {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed || (!trimmed.startsWith('{') && !trimmed.startsWith('['))) {
      return null
    }
    try {
      return parseJsonRecord(JSON.parse(trimmed))
    } catch {
      return null
    }
  }

  if (!value || Array.isArray(value) || typeof value !== 'object') return null
  return value as Record<string, unknown>
}

function normalizeMaterialUrl(value: unknown) {
  if (typeof value !== 'string') return null
  const trimmed = value.trim()
  const lower = trimmed.toLowerCase()
  if (
    lower.startsWith('http://') ||
    lower.startsWith('https://') ||
    trimmed.startsWith('/')
  ) {
    return trimmed
  }
  return null
}

function findMaterialUrls(value: unknown): string[] {
  const directUrl = normalizeMaterialUrl(value)
  if (directUrl) return [directUrl]

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed.startsWith('{') && !trimmed.startsWith('[')) return []
    try {
      return findMaterialUrls(JSON.parse(trimmed))
    } catch {
      return []
    }
  }

  if (Array.isArray(value)) {
    return value.flatMap(findMaterialUrls)
  }

  if (!value || typeof value !== 'object') return []
  const record = value as Record<string, unknown>
  return ['url', 'previewUrl'].flatMap((key) => findMaterialUrls(record[key]))
}

function findFirstPrompt(value: unknown): string | null {
  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return null
    if (trimmed.startsWith('{') || trimmed.startsWith('[')) {
      try {
        return findFirstPrompt(JSON.parse(trimmed))
      } catch {
        return null
      }
    }
    return null
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      const prompt = findFirstPrompt(item)
      if (prompt) return prompt
    }
    return null
  }

  if (!value || typeof value !== 'object') return null

  const record = value as Record<string, unknown>
  for (const key of ['prompt', 'input', 'text']) {
    const prompt = normalizePrompt(record[key])
    if (prompt) return prompt
    const nestedPrompt = findFirstPrompt(record[key])
    if (nestedPrompt) return nestedPrompt
  }

  for (const key of [
    'properties',
    'request',
    'body',
    'payload',
    'data',
    'messages',
    'content',
  ]) {
    const prompt = findFirstPrompt(record[key])
    if (prompt) return prompt
  }

  return null
}
