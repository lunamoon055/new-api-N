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
import type { TaskLog } from '../types'

export interface TaskFailureDetails {
  downstreamMessage: string
  upstreamRawError?: string
}

const containsHan = (value: string) =>
  /[\u3400-\u9fff\uf900-\ufaff]/u.test(value)

function findChineseErrorMessage(
  value: unknown,
  depth = 0
): string | undefined {
  if (depth > 6 || value == null) return undefined

  if (typeof value === 'string') {
    const trimmed = value.trim()
    if (!trimmed) return undefined
    if (
      trimmed.startsWith('{') ||
      trimmed.startsWith('[') ||
      trimmed.startsWith('"')
    ) {
      try {
        const nested = JSON.parse(trimmed) as unknown
        const message = findChineseErrorMessage(nested, depth + 1)
        if (message) return message
      } catch {
        // Plain provider messages may begin with punctuation; inspect as text.
      }
    }
    if (!containsHan(trimmed)) return undefined
    return trimmed.replace(/^(?:failed|failure|error)\s*:\s*/i, '').trim()
  }

  if (Array.isArray(value)) {
    for (const item of value) {
      const message = findChineseErrorMessage(item, depth + 1)
      if (message) return message
    }
    return undefined
  }

  if (typeof value === 'object') {
    const record = value as Record<string, unknown>
    for (const key of [
      'message',
      'msg',
      'detail',
      'reason',
      'error',
      'status',
    ]) {
      const message = findChineseErrorMessage(record[key], depth + 1)
      if (message) return message
    }
  }

  return undefined
}

function normalizeChineseUpstreamMessage(message: string): string {
  if (
    [
      '积分不足',
      '积分不够',
      '点数不足',
      '额度不足',
      '配额不足',
      '余额不足',
      '余额不够',
    ].some((keyword) => message.includes(keyword))
  ) {
    return '积分不足，请联系管理员'
  }
  return message
}

export function getTaskFailureDetails(
  log: Pick<TaskLog, 'fail_reason' | 'raw_fail_reason'>,
  canViewPrivateTaskDetails: boolean
): TaskFailureDetails {
  let downstreamMessage = log.fail_reason?.trim() ?? ''
  const upstreamRawError = canViewPrivateTaskDetails
    ? log.raw_fail_reason?.trim() || undefined
    : undefined

  // Defend against cached or historical API rows that still pair a generic
  // translation with an already-readable Chinese provider message.
  if (upstreamRawError) {
    const chineseMessage = findChineseErrorMessage(upstreamRawError)
    if (chineseMessage) {
      downstreamMessage = normalizeChineseUpstreamMessage(chineseMessage)
    }
  }

  return { downstreamMessage, upstreamRawError }
}
