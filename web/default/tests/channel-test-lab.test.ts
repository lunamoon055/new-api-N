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
  getChannelTestPreviewPayload,
  getChannelTestTemplate,
} from '../src/features/channels/lib/channel-test-lab'

describe('channel test lab helpers', () => {
  it('builds an image generation payload for gpt-image2', () => {
    expect(
      getChannelTestPreviewPayload('image-generation', 'gpt-image2')
    ).toEqual({
      model: 'gpt-image2',
      prompt: 'a cute cat',
      output_resolution: '1K',
      aspect_ratio: '1:1',
    })
  })

  it('marks async video tests as non-streaming', () => {
    expect(getChannelTestTemplate('openai-video-async').supportsStream).toBe(
      false
    )
  })

  it('uses the selected model in the auto payload preview', () => {
    expect(getChannelTestPreviewPayload('auto', 'custom-model')).toMatchObject({
      model: 'custom-model',
    })
  })

  it('uses the standard OpenAI videos payload by default', () => {
    expect(
      getChannelTestPreviewPayload('openai-video', 'seedance2(933)')
    ).toMatchObject({
      model: 'seedance2(933)',
      size: '1280x720',
      seconds: '4',
    })
  })
})
