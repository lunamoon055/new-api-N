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
import type { CreationAsset, CreationMode, CreationResult } from './types'

export type CreationResolution = '1080p' | '2k' | '4k'
export type CreationDuration = '4' | '5' | '8' | '10' | '12' | '15'

export type CreationVideoOptions = {
  resolution: CreationResolution
  duration: CreationDuration
}

export type CreationVideoRequestOptions = {
  seconds: string
  size: string
  estimateSeconds: number
}

export type CreationHistoryItem = {
  id: string
  createdAt: number
  mode: CreationMode
  model: string
  prompt: string
  assets?: CreationAsset[]
  result: CreationResult
  videoOptions?: CreationVideoOptions
}

export type CreationHistoryStorage = Pick<
  Storage,
  'getItem' | 'removeItem' | 'setItem'
>

export const CREATION_HISTORY_LIMIT = 20

export const CREATION_RESOLUTION_OPTIONS: Array<{
  value: CreationResolution
  label: string
  size: string
  estimateMultiplier: number
}> = [
  { value: '1080p', label: '1080', size: '1920x1080', estimateMultiplier: 1 },
  { value: '2k', label: '2K', size: '2560x1440', estimateMultiplier: 1.35 },
  { value: '4k', label: '4K', size: '3840x2160', estimateMultiplier: 1.75 },
]

export const CREATION_DURATION_OPTIONS: Array<{
  value: CreationDuration
  label: string
  seconds: string
  estimateSeconds: number
}> = [
  { value: '5', label: '5s', seconds: '5', estimateSeconds: 90 },
  { value: '10', label: '10s', seconds: '10', estimateSeconds: 150 },
  { value: '15', label: '15s', seconds: '15', estimateSeconds: 210 },
]

export const SORA2_CREATION_DURATION_OPTIONS: Array<{
  value: CreationDuration
  label: string
  seconds: string
  estimateSeconds: number
}> = [
  { value: '4', label: '4s', seconds: '4', estimateSeconds: 75 },
  { value: '8', label: '8s', seconds: '8', estimateSeconds: 135 },
  { value: '12', label: '12s', seconds: '12', estimateSeconds: 195 },
]

export const DEFAULT_CREATION_VIDEO_OPTIONS: CreationVideoOptions = {
  resolution: '1080p',
  duration: '5',
}

const SORA2_VIDEO_SIZE = '720x1280'

function isSora2Model(modelId?: string) {
  const normalized = modelId?.trim().toLowerCase()
  return normalized === 'sora2' || normalized === 'sora-2'
}

export function getCreationDurationOptions(modelId?: string) {
  return isSora2Model(modelId)
    ? SORA2_CREATION_DURATION_OPTIONS
    : CREATION_DURATION_OPTIONS
}

export function normalizeCreationVideoOptions(
  options: CreationVideoOptions,
  modelId?: string
): CreationVideoOptions {
  const resolution =
    CREATION_RESOLUTION_OPTIONS.find(
      (item) => item.value === options.resolution
    ) ?? CREATION_RESOLUTION_OPTIONS[0]
  const durations = getCreationDurationOptions(modelId)
  const duration =
    durations.find((item) => item.value === options.duration) ?? durations[0]

  return {
    resolution: resolution.value,
    duration: duration.value,
  }
}

export function getCreationVideoRequestOptions(
  options: CreationVideoOptions,
  modelId?: string
): CreationVideoRequestOptions {
  const normalizedOptions = normalizeCreationVideoOptions(options, modelId)
  const resolution =
    CREATION_RESOLUTION_OPTIONS.find(
      (item) => item.value === normalizedOptions.resolution
    ) ?? CREATION_RESOLUTION_OPTIONS[0]
  const duration =
    getCreationDurationOptions(modelId).find(
      (item) => item.value === normalizedOptions.duration
    ) ?? getCreationDurationOptions(modelId)[0]

  return {
    seconds: duration.seconds,
    size: isSora2Model(modelId) ? SORA2_VIDEO_SIZE : resolution.size,
    estimateSeconds: Math.ceil(
      duration.estimateSeconds *
        (isSora2Model(modelId) ? 1 : resolution.estimateMultiplier)
    ),
  }
}

export function getCreationHistoryStorageKey(userId?: number | null) {
  return `creation-center-history:${userId ?? 'guest'}`
}

export function loadCreationHistory(
  storage: CreationHistoryStorage,
  key: string
): CreationHistoryItem[] {
  try {
    const raw = storage.getItem(key)
    if (!raw) return []
    const parsed = JSON.parse(raw)
    if (!Array.isArray(parsed)) return []
    return parsed
      .filter(isCreationHistoryItem)
      .sort((left, right) => right.createdAt - left.createdAt)
      .slice(0, CREATION_HISTORY_LIMIT)
  } catch {
    storage.removeItem(key)
    return []
  }
}

export function saveCreationHistory(
  storage: CreationHistoryStorage,
  key: string,
  items: CreationHistoryItem[]
) {
  storage.setItem(
    key,
    JSON.stringify(
      [...items]
        .filter(isCreationHistoryItem)
        .sort((left, right) => right.createdAt - left.createdAt)
        .slice(0, CREATION_HISTORY_LIMIT)
    )
  )
}

export function upsertCreationHistoryItem(
  items: CreationHistoryItem[],
  item: CreationHistoryItem
) {
  return [item, ...items.filter((current) => current.id !== item.id)]
    .sort((left, right) => right.createdAt - left.createdAt)
    .slice(0, CREATION_HISTORY_LIMIT)
}

export function getCreationCountdownSeconds(
  createdAt: number | undefined,
  estimateSeconds: number | undefined,
  now = Date.now()
) {
  if (!createdAt || !estimateSeconds) return 0
  return Math.max(
    0,
    Math.ceil((createdAt + estimateSeconds * 1000 - now) / 1000)
  )
}

export function getCreationTimedOut(
  createdAt: number | undefined,
  estimateSeconds: number | undefined,
  now = Date.now()
) {
  return (
    !!createdAt &&
    !!estimateSeconds &&
    getCreationCountdownSeconds(createdAt, estimateSeconds, now) <= 0
  )
}

export function formatCreationCountdown(seconds: number) {
  const safeSeconds = Math.max(0, Math.floor(seconds))
  const minutes = Math.floor(safeSeconds / 60)
  const rest = safeSeconds % 60
  return `${minutes.toString().padStart(2, '0')}:${rest
    .toString()
    .padStart(2, '0')}`
}

export function composeCreationPrompt(prompt: string, assets: CreationAsset[]) {
  const trimmedPrompt = prompt.trim()
  const cleanAssets = assets.filter((asset) => asset.name.trim())
  if (!cleanAssets.length) return trimmedPrompt

  return [
    trimmedPrompt,
    '',
    '参考素材 / Reference assets:',
    ...cleanAssets.flatMap((asset, index) => {
      const lines = [
        `${index + 1}. ${asset.name}${asset.type ? ` (${asset.type})` : ''}`,
      ]
      if (asset.text) {
        lines.push(asset.text.slice(0, 2000))
      }
      return lines
    }),
  ].join('\n')
}

function isCreationHistoryItem(value: unknown): value is CreationHistoryItem {
  if (!value || typeof value !== 'object') return false
  const item = value as Partial<CreationHistoryItem>
  return (
    typeof item.id === 'string' &&
    typeof item.createdAt === 'number' &&
    typeof item.model === 'string' &&
    typeof item.prompt === 'string' &&
    !!item.result &&
    typeof item.result === 'object' &&
    ['chat', 'image', 'video'].includes(item.mode ?? '')
  )
}
