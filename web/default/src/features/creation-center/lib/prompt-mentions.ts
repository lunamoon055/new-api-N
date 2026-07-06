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

export type PromptMentionTrigger = {
  start: number
  end: number
  query: string
}

export function getPromptMentionTrigger(
  value: string,
  caretPosition: number | null | undefined
): PromptMentionTrigger | null {
  const end = caretPosition ?? value.length
  const prefix = value.slice(0, end)
  const start = prefix.lastIndexOf('@')
  if (start < 0) return null
  const query = prefix.slice(start + 1)
  if (query.includes('@') || /\s/.test(query)) return null
  return { start, end, query }
}
