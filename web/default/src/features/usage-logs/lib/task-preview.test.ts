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
  getTaskLogImagePreviewUrls,
  getTaskLogVideoPreviewUrl,
  getTaskLogInputMaterials,
  getTaskLogModelName,
  getVisibleTaskLogInputMaterials,
} from './task-preview'

describe('task log image preview URLs', () => {
  test('uses and deduplicates the persisted result URL', () => {
    assert.deepEqual(
      getTaskLogImagePreviewUrls({
        action: 'imageGenerate',
        status: 'SUCCESS',
        result_url: 'https://cdn.example.com/generated.png',
        data: JSON.stringify({
          result_url: 'https://cdn.example.com/generated.png',
          data: [{ url: 'https://cdn.example.com/fallback.webp' }],
        }),
      }),
      [
        'https://cdn.example.com/generated.png',
        'https://cdn.example.com/fallback.webp',
      ]
    )
  })

  test('reads LinkSky data-array results when result_url is absent', () => {
    assert.deepEqual(
      getTaskLogImagePreviewUrls({
        action: 'imageGenerate',
        status: 'SUCCESS',
        data: {
          task_id: 'upstream-task',
          status: 'completed',
          data: [{ url: '/api/image-results/generated.png' }],
        },
      }),
      ['/api/image-results/generated.png']
    )
  })

  test('does not expose unfinished or non-image task results as images', () => {
    const result = {
      result_url: 'https://cdn.example.com/generated.png',
      data: null,
    }
    assert.deepEqual(
      getTaskLogImagePreviewUrls({
        ...result,
        action: 'imageGenerate',
        status: 'IN_PROGRESS',
      }),
      []
    )
    assert.deepEqual(
      getTaskLogImagePreviewUrls({
        ...result,
        action: 'generate',
        status: 'SUCCESS',
      }),
      []
    )
  })
})

describe('task log video preview URL', () => {
  test('prefers the transferred media URL', () => {
    assert.equal(
      getTaskLogVideoPreviewUrl({
        action: 'generate',
        status: 'SUCCESS',
        task_id: 'task_public',
        result_url: 'https://media.example/video.mp4',
        data: JSON.stringify({
          url: 'https://upstream.example/v1/videos/task_upstream/content',
        }),
      }),
      'https://media.example/video.mp4'
    )
  })

  test('uses the protected site proxy for an upstream content URL with a different task ID', () => {
    assert.equal(
      getTaskLogVideoPreviewUrl({
        action: 'generate',
        status: 'SUCCESS',
        task_id: 'task_public',
        result_url: 'https://upstream.example/v1/videos/task_upstream/content',
        data: null,
      }),
      '/v1/videos/task_public/content'
    )
  })

  test('recognizes async-generations content URLs as protected upstream URLs', () => {
    assert.equal(
      getTaskLogVideoPreviewUrl({
        action: 'generate',
        status: 'SUCCESS',
        task_id: 'task_public',
        result_url:
          'https://upstream.example/v1/video/async-generations/task_upstream/content',
        data: null,
      }),
      '/v1/videos/task_public/content'
    )
  })
})

describe('task log model name', () => {
  test('prefers the model name returned by the task DTO', () => {
    assert.equal(
      getTaskLogModelName({
        model_name: 'Seedance-2.5',
        properties: { origin_model_name: 'seedance-2.5' },
      }),
      'Seedance-2.5'
    )
  })

  test('falls back to persisted origin and upstream model names', () => {
    assert.equal(
      getTaskLogModelName({
        properties: JSON.stringify({
          origin_model_name: 'videos-4',
          upstream_model_name: 'videos-standard',
        }),
      }),
      'videos-4'
    )
    assert.equal(
      getTaskLogModelName({
        properties: { upstream_model_name: 'seedance-2.5' },
      }),
      'seedance-2.5'
    )
  })
})

describe('task log input materials', () => {
  test('extracts persisted image, video, and audio links in submission order', () => {
    assert.deepEqual(
      getTaskLogInputMaterials({
        properties: {
          input_images: [
            'https://cdn.example.com/reference.png',
            '/api/creation/reference-files/local-image',
          ],
          input_videos: ['https://cdn.example.com/reference.mp4'],
          input_audios: ['https://cdn.example.com/reference.wav'],
        },
      }),
      [
        { kind: 'image', url: 'https://cdn.example.com/reference.png' },
        {
          kind: 'image',
          url: '/api/creation/reference-files/local-image',
        },
        { kind: 'video', url: 'https://cdn.example.com/reference.mp4' },
        { kind: 'audio', url: 'https://cdn.example.com/reference.wav' },
      ]
    )
  })

  test('supports raw request shapes without treating outputs or inline data as inputs', () => {
    assert.deepEqual(
      getTaskLogInputMaterials({
        properties: JSON.stringify({
          request: {
            image_url: 'data:image/png;base64,AAAA',
            image_urls: ['https://cdn.example.com/frame.png'],
            video_reference: [{ url: 'https://cdn.example.com/reference.mp4' }],
            audio_url: 'https://cdn.example.com/reference.mp3',
            result_url: 'https://cdn.example.com/generated.mp4',
          },
        }),
      }),
      [
        { kind: 'image', url: 'https://cdn.example.com/frame.png' },
        { kind: 'video', url: 'https://cdn.example.com/reference.mp4' },
        { kind: 'audio', url: 'https://cdn.example.com/reference.mp3' },
      ]
    )
  })

  test('only returns input materials when the viewer has permission', () => {
    const log = {
      properties: {
        input_images: ['https://cdn.example.com/reference.png'],
      },
    }

    assert.deepEqual(getVisibleTaskLogInputMaterials(log, false), [])
    assert.deepEqual(getVisibleTaskLogInputMaterials(log, true), [
      { kind: 'image', url: 'https://cdn.example.com/reference.png' },
    ])
  })
})
