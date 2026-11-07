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
import { t } from 'i18next'

// The edit contract accepts files, while the reference picker stores URLs.
// Both creation surfaces share this conversion and send no reference URLs in
// a generation request that would otherwise silently ignore them.
export async function prepareImageTransport(
  endpoint: string,
  payload: Record<string, unknown>,
  signal?: AbortSignal
): Promise<{ endpoint: string; payload: Record<string, unknown> | FormData }> {
  const references = payload['image[]']
  if (
    endpoint !== '/api/creation/images/generations' ||
    !Array.isArray(references) ||
    references.length === 0
  )
    return { endpoint, payload }

  if (references.length > 20) throw new Error(t('Request failed'))
  const form = new FormData()
  for (const [key, value] of Object.entries(payload)) {
    if (key !== 'image[]' && value !== undefined && value !== null)
      form.append(key, String(value))
  }
  for (const reference of references) {
    if (typeof reference !== 'string') throw new Error(t('Request failed'))
    const response = await fetch(reference, { signal, credentials: 'omit' })
    if (!response.ok) throw new Error(t('Request failed'))
    const blob = await response.blob()
    if (
      blob.size > 50 * 1024 * 1024 ||
      !['image/png', 'image/jpeg', 'image/webp'].includes(blob.type)
    )
      throw new Error(t('Request failed'))
    const extension = blob.type.split('/')[1]
    form.append(
      'image[]',
      blob,
      `reference-${form.getAll('image[]').length + 1}.${extension}`
    )
  }
  return { endpoint: '/api/creation/images/edits', payload: form }
}
