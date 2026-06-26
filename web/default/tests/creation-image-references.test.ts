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
  EMPTY_CREATION_IMAGE_REFERENCES,
  getCreationImageReferenceError,
  getCreationImageRequestOptions,
  normalizeCreationImageReferences,
  supportsCreationImageReferences,
} from '../src/features/creation-center/session'

describe('gpt-image2 creation references', () => {
  it('enables reference images only for gpt-image2', () => {
    expect(supportsCreationImageReferences('gpt-image2')).toBe(true)
    expect(supportsCreationImageReferences('GPT-IMAGE2')).toBe(true)
    expect(supportsCreationImageReferences('gpt-image-1')).toBe(false)
  })

  it('validates gpt-image2 image reference limits and formats', () => {
    expect(
      getCreationImageReferenceError('gpt-image2', {
        imageUrls: Array.from(
          { length: 7 },
          (_, index) => `https://cdn.example/${index}.png`
        ),
      })
    ).toBe('Gpt-image2 accepts at most 6 reference images.')
    expect(
      getCreationImageReferenceError('gpt-image2', {
        imageUrls: ['data:image/bmp;base64,AAAA'],
      })
    ).toBe('Reference image format must be PNG, JPEG, WebP, GIF, or AVIF.')
    expect(
      getCreationImageReferenceError('gpt-image2', {
        imageUrls: ['data:text/plain;base64,AAAA'],
      })
    ).toBe('Reference images must be images or HTTP URLs.')
  })

  it('maps uploaded image references to messages content', () => {
    expect(
      getCreationImageRequestOptions('make it cinematic', 'gpt-image2', {
        imageUrls: [
          {
            url: 'https://cdn.example/source.png',
            previewUrl: 'blob:http://localhost:3001/local-preview',
          },
          'https://cdn.example/style.webp',
        ],
      })
    ).toEqual({
      output_resolution: '1K',
      aspect_ratio: '1:1',
      messages: [
        {
          role: 'user',
          content: [
            { type: 'text', text: 'make it cinematic' },
            {
              type: 'image_url',
              image_url: { url: 'https://cdn.example/source.png' },
            },
            {
              type: 'image_url',
              image_url: { url: 'https://cdn.example/style.webp' },
            },
          ],
        },
      ],
    })
  })

  it('keeps text-only gpt-image2 payload fields fixed', () => {
    expect(
      getCreationImageRequestOptions(
        'future city poster',
        'gpt-image2',
        EMPTY_CREATION_IMAGE_REFERENCES
      )
    ).toEqual({
      output_resolution: '1K',
      aspect_ratio: '1:1',
    })
  })

  it('drops unsupported image references for other image models', () => {
    expect(
      normalizeCreationImageReferences(
        { imageUrls: ['https://cdn.example/source.png'] },
        'gpt-image-1'
      )
    ).toEqual(EMPTY_CREATION_IMAGE_REFERENCES)
    expect(
      getCreationImageRequestOptions('prompt', 'gpt-image-1', {
        imageUrls: ['https://cdn.example/source.png'],
      })
    ).toEqual({})
  })
})
