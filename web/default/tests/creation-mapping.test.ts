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
import type { CreationModel } from '../src/features/creation-center/types'
import { buildCreationVideoSubmitRequest } from '../src/features/creation-center/video-request'
import {
  getCreationImageRequestOptions,
  getCreationImageReferenceLimits,
  usesAsyncCreationImageModel,
} from '../src/features/media-generation/image-options'
import { getCreationVideoCapabilities } from '../src/features/media-generation/video-options'
import {
  getXJZVideoModelSpec,
  XJZ_VIDEO_ALIASES,
} from '../src/features/media-generation/xjz-video-models'

describe('mapped creation models', () => {
  it('selects every supported video route from the mapped protocol', () => {
    for (const [path, endpoint] of [
      ['/v1/videos', '/api/creation/videos'],
      ['/v1/video/generations', '/api/creation/video/generations'],
      ['/v1/video/async-generations', '/api/creation/video/async-generations'],
    ]) {
      const request = buildCreationVideoSubmitRequest({
        model: {
          id: 'opaque alias',
          supported_endpoint_types: ['openai-video'],
          mapped_endpoints: [
            { type: 'openai-video', kind: 'video', path, method: 'POST' },
          ],
        },
        prompt: 'demo',
      })
      expect(request.endpoint).toBe(endpoint)
      expect(request.payload).toEqual({ model: 'opaque alias', prompt: 'demo' })
    }
  })

  it('retains arbitrary public IDs and uses the mapped video fields and limits', () => {
    const model: CreationModel = {
      id: 'my video alias',
      supported_endpoint_types: ['openai-video'],
      mapped_endpoints: [
        {
          type: 'openai-video',
          kind: 'video',
          path: '/v1/videos',
          method: 'POST',
          parameters: {
            durations: [30],
            resolutions: ['720p'],
            ratios: ['9:16'],
            duration_field: 'duration',
            ratio_field: 'ratio',
            resolution_field: 'resolution',
            request_resolution: '720P',
            image_field: 'referenceImages',
            max_images: 10,
            max_videos: 0,
            max_audios: 0,
            allow_frames: true,
          },
        },
      ],
    }
    expect(getCreationVideoCapabilities(model)?.durations).toEqual(['30'])
    const request = buildCreationVideoSubmitRequest({
      model,
      prompt: 'demo',
      videoOptions: { duration: '6', resolution: '480p', aspectRatio: '16:9' },
      videoReferences: {
        referenceMode: 'image',
        imageUrls: ['https://cdn.example/a.jpg'],
        videoUrls: [],
        audioUrls: [],
        audioUrl: '',
        startImageUrl: '',
        endImageUrl: '',
      },
    })
    expect(request.endpoint).toBe('/api/creation/videos')
    expect(request.payload).toMatchObject({
      model: 'my video alias',
      duration: 30,
      ratio: '9:16',
      resolution: '720P',
      referenceImages: ['https://cdn.example/a.jpg'],
    })
    expect(request.payload).not.toHaveProperty('image_urls')
    model.mapped_endpoints![0].parameters!.min_images = 1
    expect(getCreationVideoCapabilities(model)?.referenceModes).toEqual([
      'image',
      'frames',
    ])
  })

  it('mapped images use their own resolution and reference fields', () => {
    const model: CreationModel = {
      id: 'my image alias',
      supported_endpoint_types: ['image-generation'],
      mapped_endpoints: [
        {
          type: 'image-generation',
          kind: 'image',
          path: '/v1/images/generations',
          method: 'POST',
          parameters: {
            resolutions: ['1K', '2K'],
            ratios: ['9:16'],
            ratio_field: 'aspect_ratio',
            resolution_field: 'output_resolution',
            image_field: 'images',
            max_images: 17,
            max_videos: 0,
            max_audios: 0,
          },
        },
      ],
    }
    expect(getCreationImageReferenceLimits(model).maxImages).toBe(17)
    expect(
      getCreationImageRequestOptions(
        'demo',
        model,
        { imageUrls: ['https://cdn.example/a.jpg'] },
        { aspectRatio: '9:16', outputResolution: '2K' }
      )
    ).toEqual({
      aspect_ratio: '9:16',
      output_resolution: '2K',
      images: ['https://cdn.example/a.jpg'],
    })
    expect(usesAsyncCreationImageModel(model)).toBe(false)
    model.mapped_endpoints![0].path = '/v1/images/async-generations'
    expect(usesAsyncCreationImageModel(model)).toBe(true)
  })

  it('keeps exact supplier suffix IDs in outgoing requests', () => {
    expect(Object.keys(XJZ_VIDEO_ALIASES)).toHaveLength(10)
    for (const [id, target] of Object.entries(XJZ_VIDEO_ALIASES)) {
      const spec = getXJZVideoModelSpec(target)!
      const request = buildCreationVideoSubmitRequest({
        model: { id, supported_endpoint_types: ['openai-video'] },
        prompt: 'demo',
        videoOptions: {
          duration: spec.durations[0],
          resolution: spec.resolutions[0],
          aspectRatio: spec.ratios[0],
        },
      })
      expect(request.endpoint).toBe('/api/creation/videos')
      expect(request.payload.model).toBe(id)
      expect(request.payload.duration).toBe(Number(spec.durations[0]))
    }
    expect(getXJZVideoModelSpec('grok-1.5-700-custom')).toBeUndefined()
  })
})
