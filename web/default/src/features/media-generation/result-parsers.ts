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

export type MediaTaskStatus =
  | 'queued'
  | 'processing'
  | 'completed'
  | 'failed'
  | 'unknown'

export type ParsedImageGenerationResult = {
  id?: string
  taskId?: string
  status?: MediaTaskStatus
  imageUrl?: string
  revisedPrompt?: string
}

export type ParsedVideoGenerationResult = {
  id?: string
  taskId?: string
  status: MediaTaskStatus
  upstreamStatus?: string
  videoUrl?: string
}

export function parseImageGenerationResult(
  raw: unknown
): ParsedImageGenerationResult {
  const data = asRecord(raw)
  const envelopeData = asRecord(data.data)
  const source =
    Object.keys(envelopeData).length && !Array.isArray(data.data)
      ? envelopeData
      : data
  const firstImage = Array.isArray(data.data) ? asRecord(data.data[0]) : {}
  const sourceData = asRecord(source.data)
  const nestedData = asRecord(sourceData.data)
  const b64 = getString(firstImage, 'b64_json')
  const imageUrl =
    getString(firstImage, 'url') ||
    getImageURL(source) ||
    getImageURL(sourceData) ||
    getImageURL(nestedData) ||
    (b64 ? `data:image/png;base64,${b64}` : undefined)
  const status = getString(source, 'status') || getString(data, 'status')
  const taskId =
    getString(source, 'task_id') ||
    getString(data, 'task_id') ||
    getString(source, 'id') ||
    getString(data, 'id')

  return {
    id: getString(source, 'id') || getString(data, 'id'),
    taskId,
    status: normalizeMediaTaskStatus(status),
    imageUrl,
    revisedPrompt: getString(firstImage, 'revised_prompt'),
  }
}

export function parseVideoGenerationResult(
  raw: unknown
): ParsedVideoGenerationResult {
  const data = asRecord(raw)
  const envelopeData = asRecord(data.data)
  const source = Object.keys(envelopeData).length ? envelopeData : data
  const metadata = asRecord(source.metadata)
  const status = getString(source, 'status') || getString(data, 'status')
  const taskId =
    getString(source, 'task_id') ||
    getString(data, 'task_id') ||
    getString(source, 'id') ||
    getString(data, 'id')

  return {
    id: getString(source, 'id') || getString(data, 'id'),
    taskId,
    status: normalizeMediaTaskStatus(status),
    upstreamStatus: status,
    videoUrl:
      firstHTTPURL(
        source,
        'url',
        'object',
        'result_url',
        'output_url',
        'video_url',
        'content_url'
      ) ||
      firstHTTPURL(
        metadata,
        'url',
        'result_url',
        'output_url',
        'video_url',
        'content_url'
      ),
  }
}

export function normalizeMediaTaskStatus(
  status: string | undefined
): MediaTaskStatus {
  const normalizedStatus = status?.trim().toLowerCase()
  if (normalizedStatus?.startsWith('failed:')) return 'failed'

  switch (normalizedStatus) {
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

export function extractMediaErrorMessage(raw: unknown): string | undefined {
  const data = asRecord(raw)
  const envelopeData = asRecord(data.data)
  const error = asRecord(data.error)
  const envelopeError = asRecord(envelopeData.error)
  const status = getString(envelopeData, 'status') || getString(data, 'status')
  const failedStatus = status?.match(/^failed:\s*(.+)$/i)?.[1]?.trim()
  return (
    getString(error, 'message') ||
    getString(envelopeError, 'message') ||
    getString(data, 'message') ||
    failedStatus
  )
}

export function getString(data: Record<string, unknown>, key: string) {
  const value = data[key]
  return typeof value === 'string' && value.trim() ? value : undefined
}

export function asRecord(value: unknown): Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
    ? (value as Record<string, unknown>)
    : {}
}

function firstHTTPURL(data: Record<string, unknown>, ...keys: string[]) {
  for (const key of keys) {
    const value = getString(data, key)
    if (value && /^https?:\/\/[^\s]+$/i.test(value)) return value
  }
  return undefined
}

function getImageURL(data: Record<string, unknown>) {
  return (
    getString(data, 'result_url') ||
    getString(data, 'image_url') ||
    getString(data, 'url') ||
    getString(data, 'output_url') ||
    getFirstImageArrayURL(data.images) ||
    getFirstImageArrayURL(data.data)
  )
}

function getFirstImageArrayURL(value: unknown) {
  if (!Array.isArray(value)) return undefined

  for (const item of value) {
    if (typeof item === 'string' && item.trim()) return item
    const image = asRecord(item)
    const url =
      getString(image, 'url') ||
      getString(image, 'image_url') ||
      getString(image, 'download_url')
    if (url) return url
  }

  return undefined
}
