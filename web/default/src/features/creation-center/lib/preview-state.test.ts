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
import type { CreationResult } from '../types'
import { getVideoGenerationWaitingPhase } from './preview-state'

function createVideoResult(
  status: CreationResult['status'],
  videoUrl?: string
): CreationResult {
  return {
    mode: 'video',
    model: 'videos-4-fast',
    status,
    videoUrl,
  }
}

describe('Creation Center video waiting preview', () => {
  test('starts immediately while a video request is being submitted', () => {
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: true,
      }),
      'submitting'
    )
  })

  test('continues for queued and processing video tasks without a result', () => {
    for (const status of ['queued', 'processing'] as const) {
      assert.equal(
        getVideoGenerationWaitingPhase({
          mode: 'video',
          submitting: false,
          result: createVideoResult(status),
        }),
        status
      )
    }
  })

  test('treats an async task without an upstream status as queued', () => {
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: false,
        result: {
          ...createVideoResult('unknown'),
          taskId: 'task-1',
        },
      }),
      'queued'
    )
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: false,
        result: createVideoResult('unknown'),
      }),
      null
    )
  })

  test('does not cover a returned video or terminal task state', () => {
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: false,
        result: createVideoResult('processing', 'https://cdn.test/video.mp4'),
      }),
      null
    )
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: false,
        result: createVideoResult('completed'),
      }),
      null
    )
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'video',
        submitting: false,
        result: createVideoResult('failed'),
      }),
      null
    )
  })

  test('leaves non-video submissions unchanged', () => {
    assert.equal(
      getVideoGenerationWaitingPhase({
        mode: 'image',
        submitting: true,
      }),
      null
    )
  })
})
