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
import { describe, test } from 'node:test'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundMediaEndpoint,
  parsePlaygroundExtraBody,
  parsePlaygroundMediaResult,
} from './media-routing'
import type { Message } from '../types'

const messages: Message[] = [
  {
    key: 'prompt',
    from: 'user',
    versions: [{ id: 'v1', content: 'a cat in space' }],
  },
]

describe('Playground media testing', () => {
  test('manual image mode routes a newly added model to the selected endpoint', () => {
    assert.equal(
      getPlaygroundMediaEndpoint('new-image-model', 'image', 'async'),
      '/api/creation/images/async-generations'
    )
    assert.deepEqual(
      buildPlaygroundMediaRequest(
        'new-image-model',
        messages,
        'image',
        'async',
        {
          aspect_ratio: '16:9',
          prompt: 'ignored',
        }
      ),
      {
        model: 'new-image-model',
        prompt: 'a cat in space',
        aspect_ratio: '16:9',
      }
    )
  })

  test('manual video mode parses task results for unknown model names', () => {
    assert.equal(
      getPlaygroundMediaEndpoint('new-video-model', 'video'),
      '/api/creation/video/async-generations'
    )
    const result = parsePlaygroundMediaResult(
      { id: 'task_123', status: 'queued' },
      'new-video-model',
      'video'
    )
    assert.equal(result.mode, 'video')
    assert.equal(result.taskId, 'task_123')
  })

  test('video endpoint selection switches between standard and async task APIs', () => {
    assert.equal(
      getPlaygroundMediaEndpoint(
        'new-video-model',
        'video',
        'auto',
        'standard'
      ),
      '/api/creation/videos'
    )
    assert.equal(
      getPlaygroundMediaEndpoint('new-video-model', 'video', 'auto', 'async'),
      '/api/creation/video/async-generations'
    )
    assert.equal(
      getPlaygroundMediaEndpoint('new-video-model', 'video', 'auto', 'auto'),
      '/api/creation/video/async-generations'
    )
  })

  test('extra parameters require an object', () => {
    assert.deepEqual(parsePlaygroundExtraBody('{"duration":5}'), {
      duration: 5,
    })
    assert.throws(() => parsePlaygroundExtraBody('[]'))
    assert.throws(() => parsePlaygroundExtraBody('{invalid'))
  })
})
