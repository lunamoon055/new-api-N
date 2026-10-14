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
  getCreationVideoCapabilities,
  getCreationVideoReferenceError,
  type CreationVideoReferences,
} from '../src/features/media-generation/video-options'
import {
  XJZ_VIDEO_MODELS,
  getXJZVideoModelSpec,
} from '../src/features/media-generation/xjz-video-models'
import {
  buildPlaygroundMediaRequest,
  getPlaygroundMediaEndpoint,
  getPlaygroundModelMode,
} from '../src/features/playground/lib/media-routing'
import {
  buildCreationVideoSubmitRequest,
  getCreationVideoEndpoint,
} from '../src/features/creation-center/video-request'
import type { Message } from '../src/features/playground/types'

const prompt: Message = {
  key: 'xjz-prompt',
  from: 'user',
  versions: [{ id: 'xjz-version', content: 'a smooth camera move' }],
}

function references(
  changes: Partial<CreationVideoReferences>
): CreationVideoReferences {
  return {
    referenceMode: 'text',
    imageUrls: [],
    startImageUrl: '',
    endImageUrl: '',
    videoUrls: [],
    audioUrls: [],
    audioUrl: '',
    ...changes,
  }
}

describe('XJZ exact video models', () => {
  it('routes all documented API IDs through the standard video creation and polling path', () => {
    const modelIds = Object.keys(XJZ_VIDEO_MODELS)
    expect(modelIds).toHaveLength(38)
    for (const modelId of modelIds) {
      expect(getPlaygroundModelMode(modelId)).toBe('video')
      expect(getPlaygroundMediaEndpoint(modelId)).toBe('/api/creation/videos')
      expect(getCreationVideoEndpoint(modelId)).toBe('/api/creation/videos')
      expect(getCreationVideoCapabilities(modelId)?.kind).toBe('xjz')
    }
    expect(getXJZVideoModelSpec('特价dola-sd2.0-900')).toBeUndefined()
    expect(getCreationVideoEndpoint('future-video-model')).toBe(
      '/api/creation/video/async-generations'
    )
  })

  it('normalizes fixed 30-second playground requests without using size or aspect_ratio', () => {
    const model = '特价稳sd-2.5-30s-720P可10图'
    const request = buildPlaygroundMediaRequest(
      model,
      [prompt],
      'auto',
      'auto',
      {},
      { duration: '6', aspectRatio: '16:9' }
    )
    expect(request).toMatchObject({
      model,
      duration: 30,
      ratio: '16:9',
      resolution: '720p',
    })
    expect(request).not.toHaveProperty('seconds')
    expect(request).not.toHaveProperty('size')
    expect(request).not.toHaveProperty('aspect_ratio')
  })

  it('keeps the two similarly named 933 models on separate capabilities', () => {
    expect(getXJZVideoModelSpec('特价sd2.0-720p-933-过脸')?.durations[0]).toBe(
      '5'
    )
    expect(getXJZVideoModelSpec('特价sd2.0-933-过脸')?.durations[0]).toBe('4')
  })

  it('submits TTP-grok at a selected resolution on the standard endpoint', () => {
    const request = buildCreationVideoSubmitRequest({
      model: { id: 'TTP-grok', supported_endpoint_types: ['openai-video'] },
      prompt: 'a smooth camera move',
      videoOptions: {
        resolution: '480p',
        duration: '6',
        aspectRatio: '9:16',
      },
    })
    expect(request.endpoint).toBe('/api/creation/videos')
    expect(request.payload).toMatchObject({
      model: 'TTP-grok',
      duration: 6,
      ratio: '9:16',
      resolution: '480p',
    })

    const playgroundRequest = buildPlaygroundMediaRequest(
      'TTP-grok',
      [prompt],
      'auto',
      'async',
      {},
      { duration: '6', aspectRatio: '9:16', resolution: '480p' }
    )
    expect(playgroundRequest).toMatchObject({
      model: 'TTP-grok',
      duration: 6,
      ratio: '9:16',
      resolution: '480p',
    })
  })

  it('uses MiniMaxH3 documented image and audio fields', () => {
    const request = buildCreationVideoSubmitRequest({
      model: { id: 'MiniMaxH3', supported_endpoint_types: ['openai-video'] },
      prompt: 'a smooth camera move',
      videoOptions: {
        resolution: '2k',
        duration: '8',
        aspectRatio: '16:9',
      },
      videoReferences: references({
        referenceMode: 'image-audio',
        imageUrls: ['https://cdn.example/one.png'],
        audioUrls: ['https://cdn.example/one.mp3'],
      }),
    })
    expect(request.payload).toMatchObject({
      duration: 8,
      resolution: '2K',
      ratio: '16:9',
      image_url: 'https://cdn.example/one.png',
      audio_url: 'https://cdn.example/one.mp3',
    })
    expect(request.payload).not.toHaveProperty('reference_videos')
  })

  it('requires an image for the two Dola models before a paid request', () => {
    const model = '特价dola2.5-900-720P-固定30s'
    expect(
      getCreationVideoReferenceError(
        model,
        references({ referenceMode: 'image' })
      )
    ).toBe('Image reference mode requires at least one image reference.')
  })
})
