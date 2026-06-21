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
  const firstImage = Array.isArray(data.data) ? asRecord(data.data[0]) : {}
  const b64 = getString(firstImage, 'b64_json')
  const imageUrl =
    getString(firstImage, 'url') ||
    (b64 ? `data:image/png;base64,${b64}` : undefined)

  return {
    id: getString(data, 'id'),
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
      getString(source, 'url') ||
      getString(source, 'result_url') ||
      getString(source, 'output_url') ||
      getString(source, 'video_url') ||
      getString(metadata, 'url') ||
      getString(metadata, 'result_url') ||
      getString(metadata, 'output_url') ||
      getString(metadata, 'video_url'),
  }
}

export function normalizeMediaTaskStatus(
  status: string | undefined
): MediaTaskStatus {
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

export function extractMediaErrorMessage(raw: unknown): string | undefined {
  const data = asRecord(raw)
  const error = asRecord(data.error)
  return getString(error, 'message') || getString(data, 'message')
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
