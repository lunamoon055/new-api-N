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
import { test } from 'node:test'
import {
  extractMediaErrorMessage,
  normalizeMediaTaskStatus,
  parseVideoGenerationResult,
} from './result-parsers'

test('keeps a plain object name out of the video URL field', () => {
  const result = parseVideoGenerationResult({
    id: 'task-1',
    status: 'completed',
    object: 'video',
  })

  assert.equal(result.status, 'completed')
  assert.equal(result.videoUrl, undefined)
})

test('accepts a real object URL and extracts FAILED reason', () => {
  const result = parseVideoGenerationResult({
    id: 'task-2',
    status: 'completed',
    object: 'https://cdn.example/video.mp4',
  })
  assert.equal(result.videoUrl, 'https://cdn.example/video.mp4')

  assert.equal(normalizeMediaTaskStatus('FAILED: quota exhausted'), 'failed')
  assert.equal(
    extractMediaErrorMessage({ status: 'FAILED: quota exhausted' }),
    'quota exhausted'
  )
})
