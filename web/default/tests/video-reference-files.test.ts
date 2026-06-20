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
import { describe, expect, it } from 'bun:test'
import {
  getReferenceImageMime,
  isReferenceImageFile,
  normalizeReferenceImageDataURL,
} from '../src/features/creation-center/video-reference-files'

describe('video reference files', () => {
  it('accepts images by browser mime type or file extension fallback', () => {
    expect(isReferenceImageFile({ name: 'photo.webp', type: '' })).toBe(true)
    expect(
      isReferenceImageFile({ name: 'photo.bin', type: 'image/png' })
    ).toBe(true)
    expect(isReferenceImageFile({ name: 'notes.txt', type: '' })).toBe(false)
  })

  it('normalizes empty file reader mime prefixes to the detected image mime', () => {
    const file = { name: 'reference.jpg', type: '' }

    expect(getReferenceImageMime(file)).toBe('image/jpeg')
    expect(
      normalizeReferenceImageDataURL(file, 'data:;base64,AAAA')
    ).toBe('data:image/jpeg;base64,AAAA')
    expect(
      normalizeReferenceImageDataURL(
        { name: 'reference.png', type: 'application/octet-stream' },
        'data:application/octet-stream;base64,AAAA'
      )
    ).toBe('data:image/png;base64,AAAA')
  })
})
