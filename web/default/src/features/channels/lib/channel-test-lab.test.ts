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
    assert.deepEqual(payload.images, ['https://example.com/reference.png'])
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
    assert.equal(videoPayload.reference, 'all')
    assert.ok(Array.isArray(videoPayload.images))
    assert.ok(Array.isArray(uploadPayload.images))
    assert.equal(pollPayload.task_id, 'task_xxx')
  })
})
