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
import assert from 'node:assert/strict'
import test from 'node:test'
import { getDrawingLogImageUrls } from './drawing-results'
import type { MidjourneyLog } from '../types'

function makeLog(overrides: Partial<MidjourneyLog>): MidjourneyLog {
  return {
    id: 1,
    user_id: 1,
    channel_id: 1,
    code: 1,
    mj_id: 'request-1',
    action: 'IMAGE_GENERATION',
    submit_time: 1,
    progress: '100%',
    prompt: 'draw a cat',
    status: 'SUCCESS',
    ...overrides,
  }
}

test('collects and deduplicates all image-generation result URLs', () => {
  const urls = getDrawingLogImageUrls(
    makeLog({
      image_url: 'https://cdn.example/first.png',
      properties: JSON.stringify({
        image_urls: [
          'https://cdn.example/first.png',
          '/api/image-results/second.png',
        ],
      }),
    })
  )

  assert.deepEqual(urls, [
    'https://cdn.example/first.png',
    '/api/image-results/second.png',
  ])
})

test('keeps the primary image when legacy properties are not JSON', () => {
  assert.deepEqual(
    getDrawingLogImageUrls(
      makeLog({
        image_url: 'https://cdn.example/first.png',
        properties: 'legacy metadata',
      })
    ),
    ['https://cdn.example/first.png']
  )
})
