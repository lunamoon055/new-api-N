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
import { formatMessageForAPI, isValidMessage } from './message-utils'
import type { Message } from '../types'

export function buildPlaygroundResponsesPayload(
  model: string,
  group: string,
  messages: Message[],
  extraBody: Record<string, unknown>
) {
  return {
    ...extraBody,
    model,
    group,
    input: messages.filter(isValidMessage).map(formatMessageForAPI),
    stream: false,
  }
}

export function parsePlaygroundResponsesText(raw: unknown): string {
  if (!raw || typeof raw !== 'object') return JSON.stringify(raw) ?? ''
  const data = raw as Record<string, unknown>
  if (typeof data.output_text === 'string' && data.output_text) {
    return data.output_text
  }
  const output = Array.isArray(data.output) ? data.output : []
  const parts: string[] = []
  for (const item of output) {
    if (!item || typeof item !== 'object') continue
    const content = (item as Record<string, unknown>).content
    if (!Array.isArray(content)) continue
    for (const part of content) {
      if (!part || typeof part !== 'object') continue
      const text = (part as Record<string, unknown>).text
      if (typeof text === 'string' && text) parts.push(text)
    }
  }
  return parts.length ? parts.join('\n') : JSON.stringify(raw, null, 2)
}
