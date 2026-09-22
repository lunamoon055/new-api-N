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
  getChannelTestPreviewPayload,
  getChannelTestTemplate,
  resolveChannelTestEndpointType,
  type ChannelTestEndpointType,
} from './channel-test-lab'

describe('Sanbao channel test templates', () => {
  test('provides an editable Sanbao image generation template', () => {
    const endpointType = 'sanbao-image' as ChannelTestEndpointType

    const template = getChannelTestTemplate(endpointType)
    const payload = getChannelTestPreviewPayload(endpointType, 'image_custom')

    assert.equal(template.endpointType, endpointType)
    assert.equal(template.supportsStream, false)
    assert.equal(payload.model, 'image_custom')
    assert.equal(payload.aspect_ratio, '16:9')
    assert.deepEqual(payload.images, [])
  })

  test('provides Sanbao video, upload, and polling templates', () => {
    const videoPayload = getChannelTestPreviewPayload(
      'sanbao-video' as ChannelTestEndpointType,
      'sd2_full'
    )
    const uploadPayload = getChannelTestPreviewPayload(
      'sanbao-upload' as ChannelTestEndpointType,
      'sd2_full'
    )
    const pollPayload = getChannelTestPreviewPayload(
      'sanbao-video-poll' as ChannelTestEndpointType,
      'sd2_full'
    )

    assert.equal(videoPayload.model, 'sd2_full')
    assert.equal(videoPayload.ratio, '9:16')
    assert.equal(videoPayload.reference, undefined)
    assert.deepEqual(videoPayload.images, [])
    assert.deepEqual(uploadPayload.images, [])
    assert.equal(pollPayload.task_id, 'task_xxx')
  })
})

describe('Videos API channel test templates', () => {
  test('routes sd2 models away from the legacy async endpoint', () => {
    assert.equal(
      resolveChannelTestEndpointType('openai-video-async', 'sd2-mini'),
      'openai-video'
    )
  })

  test('uses ratio and resolution fields for sd2 models', () => {
    const payload = getChannelTestPreviewPayload('openai-video', 'sd2-mini')

    assert.deepEqual(payload, {
      model: 'sd2-mini',
      prompt: 'a mountain at sunrise',
      ratio: '16:9',
      resolution: '720p',
      duration: 5,
    })
  })

  test('keeps sd2 models on the Sanbao endpoint for Sanbao channels', () => {
    assert.equal(
      resolveChannelTestEndpointType('auto', 'sd2_full', {
        type: 58,
        base_url: 'https://sanbaobeauty.com',
      }),
      'sanbao-video'
    )
  })
})

describe('Async image channel test templates', () => {
  test('auto-detects the documented async image models', () => {
    for (const model of [
      'gpt-image-2',
      'gpt-image-2.5',
      'nano-banana-pro',
      'nano-banana2',
      'seedream-5-0',
    ]) {
      assert.equal(
        resolveChannelTestEndpointType('auto', model, {
          type: 1,
          base_url: 'https://linksky.top/',
        }),
        'openai-image-async'
      )
    }
  })

  test('keeps the same model synchronous on other channels', () => {
    assert.equal(
      resolveChannelTestEndpointType('auto', 'gpt-image-2', {
        type: 1,
        base_url: 'https://other-provider.example',
      }),
      'image-generation'
    )
  })

  test('normalizes a manual image test only for the exact LinkSky host', () => {
    assert.equal(
      resolveChannelTestEndpointType('image-generation', 'gpt-image-2', {
        type: 1,
        base_url: 'https://linksky.top/v1',
      }),
      'openai-image-async'
    )
    assert.equal(
      resolveChannelTestEndpointType('image-generation', 'gpt-image-2', {
        type: 1,
        base_url: 'https://linksky.top.evil.example',
      }),
      'image-generation'
    )
    assert.equal(
      resolveChannelTestEndpointType('image-generation', 'gpt-image-2', {
        type: 1,
        base_url: 'ftp://linksky.top',
      }),
      'image-generation'
    )
  })

  test('uses the async image request fields', () => {
    const payload = getChannelTestPreviewPayload(
      'openai-image-async',
      'gpt-image-2'
    )

    assert.deepEqual(payload, {
      model: 'gpt-image-2',
      prompt: 'a minimal poster of a sunset over the sea',
      aspect_ratio: '1:1',
      output_resolution: '2K',
    })
  })
})
