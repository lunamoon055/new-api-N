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
import fixture from '../../../common/testdata/xjz-video-models-2026-10-14.json'
import { buildCreationVideoSubmitRequest } from '../src/features/creation-center/video-request'
import {
  getCreationVideoCapabilities,
  getCreationVideoReferenceError,
  type CreationResolution,
} from '../src/features/media-generation/video-options'
import { getXJZVideoModelSpec } from '../src/features/media-generation/xjz-video-models'
import { buildPlaygroundReferences } from '../src/features/playground/lib/model-integration'
import { buildPlaygroundMediaRequest } from '../src/features/playground/lib/media-routing'
import type { Message } from '../src/features/playground/types'

const prompt: Message = {
  key: 'xjz-contract',
  from: 'user',
  versions: [{ id: 'v1', content: 'a product video' }],
}
const emptyReferences = {
  referenceMode: 'text' as const,
  imageUrls: [],
  startImageUrl: '',
  endImageUrl: '',
  videoUrls: [],
  audioUrls: [],
  audioUrl: '',
}

describe('XJZ current catalog and dedicated video contracts', () => {
  it('preserves mixed image/video references when a model has no audio input', () => {
    const request = buildCreationVideoSubmitRequest({
      model: {
        id: 'gz-sd2.5-480p',
        supported_endpoint_types: ['openai-video'],
      },
      prompt: 'demo',
      videoOptions: { resolution: '480p', duration: '30', aspectRatio: '16:9' },
      videoReferences: {
        ...emptyReferences,
        referenceMode: 'multimodal',
        imageUrls: ['https://cdn.example/a.jpg'],
        videoUrls: ['https://cdn.example/a.mp4'],
      },
    })
    expect(request.payload).toMatchObject({
      images: ['https://cdn.example/a.jpg'],
      reference_videos: ['https://cdn.example/a.mp4'],
      duration: 30,
      resolution: '480P',
    })
    expect(request.payload).not.toHaveProperty('reference_audios')
  })
  it('exposes all 25 additions with their documented capabilities and payload fields', () => {
    expect(fixture.models).toHaveLength(25)
    for (const doc of fixture.models) {
      const capability = getCreationVideoCapabilities(doc.id)
      expect(capability?.durations).toEqual(doc.durations.map(String))
      expect(capability?.resolutions).toEqual(
        doc.resolutions.map((value) => value.toLowerCase())
      )
      expect(capability?.aspectRatios).toEqual(doc.ratios)
      expect(capability?.referenceLimits).toMatchObject({
        maxImages: doc.max_images,
        maxVideos: doc.max_videos,
        maxAudios: doc.max_audios,
        maxImageSizeMB: 10,
        maxVideoSizeMB: 10,
        maxAudioSizeMB: 10,
      })
      const durations = [doc.durations[0], doc.durations.at(-1)!]
      for (const resolution of doc.resolutions) {
        for (const duration of durations) {
          const options = {
            resolution: resolution.toLowerCase() as CreationResolution,
            duration: String(duration),
            aspectRatio: doc.ratios.at(-1)!,
          }
          const request = buildCreationVideoSubmitRequest({
            model: { id: doc.id, supported_endpoint_types: ['openai-video'] },
            prompt: 'a product video',
            videoOptions: options,
          })
          expect(request.endpoint).toBe('/api/creation/videos')
          expect(request.payload).toMatchObject({
            model: doc.id,
            duration,
            [doc.ratio_field]: options.aspectRatio,
            [doc.resolution_field]: resolution,
          })
          const playground = buildPlaygroundMediaRequest(
            doc.id,
            [prompt],
            'auto',
            'auto',
            { generate_audio: false },
            options
          )
          expect(playground).toMatchObject({
            model: doc.id,
            duration,
            [doc.ratio_field]: options.aspectRatio,
            [doc.resolution_field]: resolution,
            generate_audio: false,
          })
          for (const payload of [request.payload, playground]) {
            expect(payload).not.toHaveProperty('seconds')
            expect(payload).not.toHaveProperty(
              doc.ratio_field === 'ratio' ? 'aspect_ratio' : 'ratio'
            )
            expect(payload).not.toHaveProperty(
              doc.resolution_field === 'resolution' ? 'size' : 'resolution'
            )
          }
        }
      }
    }
  })

  it('sends Viraldance 933x references as singular URL strings', () => {
    const request = buildCreationVideoSubmitRequest({
      model: {
        id: 'viraldance933x-fast',
        supported_endpoint_types: ['openai-video'],
      },
      prompt: 'demo',
      videoOptions: { duration: '8', resolution: '720p', aspectRatio: '9:16' },
      videoReferences: {
        ...emptyReferences,
        referenceMode: 'multimodal',
        imageUrls: ['https://cdn.example/a.jpg'],
        videoUrls: ['https://cdn.example/a.mp4'],
        audioUrls: ['https://cdn.example/a.mp3'],
      },
    })
    expect(request.payload).toMatchObject({
      aspect_ratio: '9:16',
      image_url: 'https://cdn.example/a.jpg',
      video_url: 'https://cdn.example/a.mp4',
      audio_url: 'https://cdn.example/a.mp3',
    })
    expect(
      buildPlaygroundReferences(
        {
          image_field: 'image_url',
          video_field: 'video_url',
          audio_field: 'audio_url',
          max_images: 1,
          max_videos: 1,
          max_audios: 1,
        },
        {
          images: 'https://cdn.example/a.jpg',
          videos: 'https://cdn.example/a.mp4',
          audios: 'https://cdn.example/a.mp3',
        }
      )
    ).toEqual({
      image_url: 'https://cdn.example/a.jpg',
      video_url: 'https://cdn.example/a.mp4',
      audio_url: 'https://cdn.example/a.mp3',
    })
  })

  it('requires an image alongside GZ audio even when a video is present', () => {
    const references = {
      ...emptyReferences,
      referenceMode: 'multimodal' as const,
      videoUrls: ['https://cdn.example/a.mp4'],
      audioUrls: ['https://cdn.example/a.mp3'],
    }
    for (const id of ['gz-sd480p', 'gz-sd720p', 'gz-sd1080p', 'gz-sd4k']) {
      expect(getCreationVideoReferenceError(id, references)).toBe(
        'Audio reference requires at least one image reference.'
      )
    }
    expect(() =>
      buildPlaygroundReferences(
        {
          audio_requires_image: true,
          image_field: 'images',
          video_field: 'reference_videos',
          audio_field: 'reference_audios',
          max_images: 9,
          max_videos: 3,
          max_audios: 3,
        },
        { videos: references.videoUrls[0], audios: references.audioUrls[0] }
      )
    ).toThrow()
  })

  it('keeps similarly named Grok and SD models isolated', () => {
    expect(getXJZVideoModelSpec('grok-1.5-700')?.durations[0]).toBe('3')
    expect(getXJZVideoModelSpec('TTP-grok')?.durations[0]).toBe('6')
    expect(
      getXJZVideoModelSpec('sd2.5-全参过真人-30-10-10')?.durations[0]
    ).toBe('5')
    expect(getXJZVideoModelSpec('gz稳定sd2.5-720全参过脸')?.durations[0]).toBe(
      '4'
    )
    expect(getXJZVideoModelSpec('gz-sd2.5-480p')?.maxAudios).toBe(0)
    expect(getXJZVideoModelSpec('grok-1.5-700-custom')).toBeUndefined()
    expect(getXJZVideoModelSpec('viraldance2.5-15')).toBeUndefined()
  })
})
