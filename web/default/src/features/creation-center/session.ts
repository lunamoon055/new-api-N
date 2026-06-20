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
import type {
  CreationVideoOptions,
  CreationVideoReferences,
} from './video-options'

export * from './video-options'

export type CreationHistoryItem = {
  id: string
  createdAt: number
  mode: CreationMode
  model: string
  prompt: string
  assets?: CreationAsset[]
  result: CreationResult
  videoOptions?: CreationVideoOptions
  videoReferences?: CreationVideoReferences
}

export type CreationHistoryStorage = Pick<
  Storage,
  'getItem' | 'removeItem' | 'setItem'
>

export const CREATION_HISTORY_LIMIT = 20

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
      .map(sanitizeCreationHistoryItem)
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
        .map(sanitizeCreationHistoryItem)
        .filter(isCreationHistoryItem)
        .sort((left, right) => right.createdAt - left.createdAt)
        .slice(0, CREATION_HISTORY_LIMIT)
    )
  )
}

export function sanitizeCreationHistoryItem(
  item: CreationHistoryItem
): CreationHistoryItem {
  const references = item.videoReferences
  if (!references) return item
  const imageUrls = Array.isArray(references.imageUrls)
    ? references.imageUrls.filter(isStorableReferenceURL)
    : []
  const startImageUrl =
    typeof references.startImageUrl === 'string' &&
    isStorableReferenceURL(references.startImageUrl)
      ? references.startImageUrl
      : ''
  const endImageUrl =
    typeof references.endImageUrl === 'string' &&
    isStorableReferenceURL(references.endImageUrl)
      ? references.endImageUrl
      : ''

  return {
    ...item,
    videoReferences: {
      ...references,
      imageUrls,
      startImageUrl,
      endImageUrl,
    },
  }
}

export function upsertCreationHistoryItem(
  items: CreationHistoryItem[],
  item: CreationHistoryItem
) {
  return [
    sanitizeCreationHistoryItem(item),
    ...items.filter((current) => current.id !== item.id),
  ]
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

function isStorableReferenceURL(value: string) {
  return /^https?:\/\//i.test(value.trim())
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
