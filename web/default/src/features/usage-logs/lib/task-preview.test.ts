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
  getTaskLogInputMaterials,
  getTaskLogModelName,
  getVisibleTaskLogInputMaterials,
} from './task-preview'

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
