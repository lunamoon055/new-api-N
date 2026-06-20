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
} from '../src/features/creation-center/video-reference-files'

describe('video reference files', () => {
  it('accepts images by browser mime type or file extension fallback', () => {
    expect(isReferenceImageFile({ name: 'photo.webp', type: '' })).toBe(true)
    expect(
      isReferenceImageFile({ name: 'photo.bin', type: 'image/png' })
    ).toBe(true)
    expect(isReferenceImageFile({ name: 'notes.txt', type: '' })).toBe(false)
  })

  it('detects the upload mime from file names when needed', () => {
    expect(getReferenceImageMime({ name: 'reference.jpg', type: '' })).toBe(
      'image/jpeg'
    )
    expect(
      getReferenceImageMime({
        name: 'reference.png',
        type: 'application/octet-stream',
      })
    ).toBe('image/png')
  })
})
