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
import type { MidjourneyLog } from '../types'

interface DrawingLogProperties {
  image_url?: unknown
  image_urls?: unknown
}

export function getDrawingLogImageUrls(log: MidjourneyLog): string[] {
  const urls: string[] = []
  const seen = new Set<string>()
  const append = (value: unknown) => {
    if (typeof value !== 'string') return
    const url = value.trim()
    if (!url || seen.has(url)) return
    seen.add(url)
    urls.push(url)
  }

  append(log.image_url)
  if (!log.properties) return urls

  try {
    const properties = JSON.parse(log.properties) as DrawingLogProperties
    append(properties.image_url)
    if (Array.isArray(properties.image_urls)) {
      properties.image_urls.forEach(append)
    }
  } catch {
    // Older task rows may contain non-JSON provider metadata.
  }

  return urls
}
